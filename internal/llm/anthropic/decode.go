package anthropic

import (
	"encoding/json"
	"fmt"

	"github.com/pdavlin/go-playball/internal/llm"
)

// anthropicEvent is the lenient envelope shared by all stream events. Only
// the fields we care about are unmarshaled.
type anthropicEvent struct {
	Type  string `json:"type"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
		// StopReason rides on the message_delta event that precedes
		// message_stop. "max_tokens" means the response was cut off at
		// the output token cap.
		StopReason string `json:"stop_reason"`
	} `json:"delta"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// decodeEvent parses one SSE event payload into an llm.Event plus an
// EventKind tag indicating whether it should propagate to the caller, end
// the stream, or be ignored. The boolean ok is false when the event has no
// JSON payload at all.
func decodeEvent(ev llm.SSEEvent) (llm.Event, llm.EventKind, bool) {
	if ev.Data == "" {
		return llm.Event{}, 0, false
	}
	var msg anthropicEvent
	if err := json.Unmarshal([]byte(ev.Data), &msg); err != nil {
		return llm.Event{
			Kind: llm.EventError,
			Err:  fmt.Errorf("anthropic decode: %w", err),
		}, llm.EventError, true
	}
	// Prefer the JSON-embedded type so an `event:` header mismatch
	// doesn't desync us.
	t := msg.Type
	if t == "" {
		t = ev.Type
	}
	switch t {
	case "content_block_delta":
		return llm.Event{Kind: llm.EventDelta, Text: msg.Delta.Text}, llm.EventDelta, true
	case "message_delta":
		// Carries the terminal stop_reason but does not end the stream
		// (message_stop does). Surface truncation via Event.Truncated so
		// runStream can latch it onto the EventDone; ok=false keeps this
		// event out of the caller's delta path.
		return llm.Event{Truncated: msg.Delta.StopReason == "max_tokens"}, 0, false
	case "message_stop":
		return llm.Event{Kind: llm.EventDone}, llm.EventDone, true
	case "error":
		errMsg := msg.Error.Message
		if errMsg == "" {
			errMsg = ev.Data
		}
		return llm.Event{
			Kind: llm.EventError,
			Err:  fmt.Errorf("anthropic: %s", errMsg),
		}, llm.EventError, true
	default:
		// message_start, content_block_start/stop, ping
		return llm.Event{}, 0, false
	}
}
