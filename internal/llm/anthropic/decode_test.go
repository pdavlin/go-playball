package anthropic

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pdavlin/go-playball/internal/llm"
)

func TestDecodeEvent_MessageDeltaMaxTokensTruncates(t *testing.T) {
	ev := llm.SSEEvent{Data: `{"type":"message_delta","delta":{"stop_reason":"max_tokens"}}`}
	got, _, ok := decodeEvent(ev)
	if ok {
		t.Error("message_delta must not propagate to the caller (ok should be false)")
	}
	if !got.Truncated {
		t.Error("stop_reason max_tokens should set Truncated")
	}
}

func TestDecodeEvent_MessageDeltaEndTurnNotTruncated(t *testing.T) {
	ev := llm.SSEEvent{Data: `{"type":"message_delta","delta":{"stop_reason":"end_turn"}}`}
	got, _, _ := decodeEvent(ev)
	if got.Truncated {
		t.Error("stop_reason end_turn must not set Truncated")
	}
}

// TestStream_TruncationLatchedOntoDone verifies that a message_delta carrying
// stop_reason max_tokens surfaces on the terminating EventDone.
func TestStream_TruncationLatchedOntoDone(t *testing.T) {
	stream := strings.Join([]string{
		`event: content_block_delta`,
		`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"Hello"}}`,
		``,
		`event: message_delta`,
		`data: {"type":"message_delta","delta":{"stop_reason":"max_tokens"}}`,
		``,
		`event: message_stop`,
		`data: {"type":"message_stop"}`,
		``,
	}, "\n")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, stream)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	ch, err := c.Stream(context.Background(), llm.Request{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}

	var sawDone, doneTruncated bool
	for _, ev := range collect(t, ch, 2*time.Second) {
		if ev.Kind == llm.EventDone {
			sawDone = true
			doneTruncated = ev.Truncated
		}
	}
	if !sawDone {
		t.Fatal("missing EventDone")
	}
	if !doneTruncated {
		t.Error("EventDone should carry Truncated after max_tokens stop")
	}
}
