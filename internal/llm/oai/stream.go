package oai

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pdavlin/go-playball/internal/llm"
)

const errBodyReadLimit = 500

// StreamChat POSTs req to url with the supplied headers and emits llm.Events
// on the returned channel. It enforces a 2xx status, decodes OpenAI-style
// SSE chunks, and treats EOF without a [DONE] sentinel as a clean
// termination (some local servers omit it).
//
// Setup errors (request build, transport failure) come back as the second
// return. Once the channel is non-nil, all subsequent failures arrive
// in-band as EventError so the UI can render them.
func StreamChat(
	ctx context.Context,
	httpC *http.Client,
	url string,
	headers map[string]string,
	req llm.Request,
	providerName string,
) (<-chan llm.Event, error) {
	body, err := buildBody(req)
	if err != nil {
		return nil, fmt.Errorf("%s: marshal request: %w", providerName, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%s: build request: %w", providerName, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := httpC.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", providerName, err)
	}

	out := make(chan llm.Event, 16)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		go func() {
			defer close(out)
			defer resp.Body.Close()
			snippet := readSnippet(resp.Body, errBodyReadLimit)
			msg := fmt.Sprintf("%s %d: %s", providerName, resp.StatusCode, snippet)
			if resp.StatusCode == http.StatusTooManyRequests {
				msg += " (try again later or pick a different model)"
			}
			out <- llm.Event{Kind: llm.EventError, Err: fmt.Errorf("%s", msg)}
		}()
		return out, nil
	}

	ctype := resp.Header.Get("Content-Type")
	if ctype != "" && !strings.Contains(ctype, "event-stream") && !strings.Contains(ctype, "text/plain") {
		go func() {
			defer close(out)
			defer resp.Body.Close()
			snippet := readSnippet(resp.Body, errBodyReadLimit)
			out <- llm.Event{
				Kind: llm.EventError,
				Err:  fmt.Errorf("%s: unexpected content-type %q: %s", providerName, ctype, snippet),
			}
		}()
		return out, nil
	}

	go runStream(ctx, resp, out, providerName)
	return out, nil
}

func runStream(ctx context.Context, resp *http.Response, out chan<- llm.Event, providerName string) {
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

	sawDone := false
	truncated := false
	err := llm.ScanSSE(resp.Body, func(ev llm.SSEEvent) bool {
		decoded, kind := decodeEvent(ev)
		// finish_reason "length" can arrive on the delta chunk or on a
		// trailing empty (skipped) chunk; latch it either way.
		if decoded.Truncated {
			truncated = true
		}
		switch kind {
		case kindDelta:
			return emit(decoded)
		case kindDone:
			sawDone = true
			emit(llm.Event{Kind: llm.EventDone, Truncated: truncated})
			return false
		}
		return true
	})
	if err != nil && ctx.Err() == nil {
		emit(llm.Event{Kind: llm.EventError, Err: fmt.Errorf("%s stream: %w", providerName, err)})
		return
	}
	// EOF without [DONE]: synthesize a clean completion.
	if !sawDone && ctx.Err() == nil {
		emit(llm.Event{Kind: llm.EventDone, Truncated: truncated})
	}
}

func readSnippet(r io.Reader, limit int) string {
	buf := make([]byte, limit)
	n, _ := io.ReadFull(r, buf)
	return strings.TrimSpace(string(buf[:n]))
}
