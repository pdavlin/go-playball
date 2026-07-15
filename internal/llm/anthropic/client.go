// Package anthropic implements an llm.Provider backed by the Anthropic
// Messages API streaming endpoint.
package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pdavlin/go-playball/internal/config"
	"github.com/pdavlin/go-playball/internal/llm"
)

const (
	defaultBaseURL   = "https://api.anthropic.com"
	anthropicVersion = "2023-06-01"
	errBodyReadLimit = 500
)

// Client is an Anthropic-backed llm.Provider.
type Client struct {
	apiKey      string
	model       string
	baseURL     string
	maxTokens   int
	temperature float64
	http        *http.Client
}

// New builds a Client from a config.Scouting. apiKey, model, and (optionally)
// base URL flow through; max_tokens defaults to 1024. A zero temperature means
// "unset" and is omitted from the request rather than defaulted.
func New(cfg config.Scouting) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1024
	}
	return &Client{
		apiKey:      cfg.APIKey,
		model:       cfg.Model,
		baseURL:     strings.TrimRight(baseURL, "/"),
		maxTokens:   maxTokens,
		temperature: cfg.Temperature,
		http:        &http.Client{},
	}
}

type msgPart struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messagesReq struct {
	Model     string    `json:"model"`
	System    string    `json:"system,omitempty"`
	Messages  []msgPart `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
	// Temperature is a pointer so an unset value is omitted entirely. Sonnet 5
	// and the Opus 4.7+ family reject any non-default sampling parameter with
	// a 400, so sending a default here would break those models.
	Temperature *float64 `json:"temperature,omitempty"`
	Stream      bool     `json:"stream"`
}

// Stream implements llm.Provider.
func (c *Client) Stream(ctx context.Context, req llm.Request) (<-chan llm.Event, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("anthropic: api key is empty")
	}
	model := req.Model
	if model == "" {
		model = c.model
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = c.maxTokens
	}
	temperature := req.Temperature
	if temperature == 0 {
		temperature = c.temperature
	}
	var temperaturePtr *float64
	if temperature != 0 {
		temperaturePtr = &temperature
	}

	var system string
	parts := make([]msgPart, 0, len(req.Messages))
	for _, m := range req.Messages {
		switch m.Role {
		case llm.RoleSystem:
			if system != "" {
				system += "\n\n"
			}
			system += m.Content
		case llm.RoleUser:
			parts = append(parts, msgPart{Role: "user", Content: m.Content})
		}
	}

	body, err := json.Marshal(messagesReq{
		Model:       model,
		System:      system,
		Messages:    parts,
		MaxTokens:   maxTokens,
		Temperature: temperaturePtr,
		Stream:      true,
	})
	if err != nil {
		return nil, fmt.Errorf("anthropic: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("anthropic: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic: %w", err)
	}

	out := make(chan llm.Event, 16)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Drain and emit a single error event in-band so the modal can
		// render it; close immediately. The caller still sees a nil
		// setup error because the stream is "open" enough to deliver
		// one message.
		go func() {
			defer close(out)
			defer resp.Body.Close()
			snippet := readSnippet(resp.Body, errBodyReadLimit)
			out <- llm.Event{
				Kind: llm.EventError,
				Err:  fmt.Errorf("anthropic %d: %s", resp.StatusCode, snippet),
			}
		}()
		return out, nil
	}

	go runStream(ctx, resp, out)
	return out, nil
}

func readSnippet(r io.Reader, limit int) string {
	buf := make([]byte, limit)
	n, _ := io.ReadFull(r, buf)
	return strings.TrimSpace(string(buf[:n]))
}

// runStream owns the HTTP response body and emits events until done.
func runStream(ctx context.Context, resp *http.Response, out chan<- llm.Event) {
	defer close(out)
	defer resp.Body.Close()

	emit := func(ev llm.Event) bool {
		select {
		case <-ctx.Done():
			return false
		case out <- ev:
			return true
		}
	}

	err := llm.ScanSSE(resp.Body, func(ev llm.SSEEvent) bool {
		decoded, kind, ok := decodeEvent(ev)
		if !ok {
			return true
		}
		switch kind {
		case llm.EventDelta:
			if decoded.Text == "" {
				return true
			}
			return emit(decoded)
		case llm.EventDone:
			emit(decoded)
			return false
		case llm.EventError:
			emit(decoded)
			return false
		}
		return true
	})
	if err != nil && ctx.Err() == nil {
		emit(llm.Event{Kind: llm.EventError, Err: fmt.Errorf("anthropic stream: %w", err)})
	}
}
