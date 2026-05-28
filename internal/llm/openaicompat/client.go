// Package openaicompat implements a generic OpenAI-compatible llm.Provider.
// Use it to point go-playball at OpenAI itself, Groq, Together, Ollama,
// LM Studio, vLLM, or any other server that speaks the OpenAI
// chat-completion streaming protocol.
package openaicompat

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/pdavlin/go-playball/internal/config"
	"github.com/pdavlin/go-playball/internal/llm"
	"github.com/pdavlin/go-playball/internal/llm/oai"
)

// Client targets {baseURL}/v1/chat/completions. Auth is optional so local
// servers without keys work without sending an empty Bearer header.
type Client struct {
	apiKey      string
	model       string
	baseURL     string
	maxTokens   int
	temperature float64
	http        *http.Client
}

// New builds a Client from a config.Scouting. BaseURL must be set by the
// caller; the factory in internal/scouting validates that before invoking
// this constructor.
func New(cfg config.Scouting) *Client {
	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1024
	}
	temperature := cfg.Temperature
	if temperature == 0 {
		temperature = 0.4
	}
	return &Client{
		apiKey:      cfg.APIKey,
		model:       cfg.Model,
		baseURL:     strings.TrimRight(cfg.BaseURL, "/"),
		maxTokens:   maxTokens,
		temperature: temperature,
		http:        &http.Client{},
	}
}

// Stream implements llm.Provider.
func (c *Client) Stream(ctx context.Context, req llm.Request) (<-chan llm.Event, error) {
	if c.baseURL == "" {
		return nil, fmt.Errorf("openai-compatible: base_url is empty")
	}
	if req.Model == "" {
		req.Model = c.model
	}
	if req.MaxTokens <= 0 {
		req.MaxTokens = c.maxTokens
	}
	if req.Temperature == 0 {
		req.Temperature = c.temperature
	}
	headers := map[string]string{}
	if c.apiKey != "" {
		headers["Authorization"] = "Bearer " + c.apiKey
	}
	endpoint := c.baseURL + "/v1/chat/completions"
	return oai.StreamChat(ctx, c.http, endpoint, headers, req, "openai-compatible")
}
