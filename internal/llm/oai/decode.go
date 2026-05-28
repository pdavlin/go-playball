// Package oai implements shared plumbing for OpenAI-compatible
// chat-completion streaming APIs. OpenRouter and arbitrary OpenAI-compatible
// servers (Ollama, LM Studio, vLLM, OpenAI itself) speak the same wire
// format, so the request shape, SSE parsing, and error handling live here.
package oai

import (
	"encoding/json"

	"github.com/pdavlin/go-playball/internal/llm"
)

// chatStreamChunk is the slice of an OpenAI streaming response we care about.
// Unknown fields are ignored.
type chatStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// decodeKind tells the caller how to react to one decoded SSE event.
type decodeKind int

const (
	// kindSkip means the event is a heartbeat or unparseable noise; keep
	// reading.
	kindSkip decodeKind = iota
	// kindDelta carries a non-empty content delta in Event.Text.
	kindDelta
	// kindDone signals the stream completed normally ([DONE] sentinel).
	kindDone
)

// decodeEvent classifies a single SSE event. The OpenAI streaming protocol
// emits one JSON object per `data:` line, with a `data: [DONE]` sentinel at
// the end. Some local servers (older Ollama) omit the sentinel and just
// close the connection; that case is handled in StreamChat via the EOF path,
// not here.
//
// Malformed JSON is treated as a skip (rather than an error) so a single bad
// chunk doesn't abort an otherwise good stream. Servers occasionally inject
// commentary or rate-limit headers as `data:` lines.
func decodeEvent(ev llm.SSEEvent) (llm.Event, decodeKind) {
	data := ev.Data
	if data == "" {
		return llm.Event{}, kindSkip
	}
	if data == "[DONE]" {
		return llm.Event{Kind: llm.EventDone}, kindDone
	}
	var chunk chatStreamChunk
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return llm.Event{}, kindSkip
	}
	if len(chunk.Choices) == 0 {
		return llm.Event{}, kindSkip
	}
	text := chunk.Choices[0].Delta.Content
	if text == "" {
		return llm.Event{}, kindSkip
	}
	return llm.Event{Kind: llm.EventDelta, Text: text}, kindDelta
}
