package oai

import (
	"testing"

	"github.com/pdavlin/go-playball/internal/llm"
)

func TestDecodeEvent_Delta(t *testing.T) {
	ev := llm.SSEEvent{Data: `{"choices":[{"delta":{"content":"Hello"}}]}`}
	got, kind := decodeEvent(ev)
	if kind != kindDelta {
		t.Fatalf("kind = %v, want kindDelta", kind)
	}
	if got.Text != "Hello" {
		t.Errorf("text = %q, want %q", got.Text, "Hello")
	}
}

func TestDecodeEvent_DoneSentinel(t *testing.T) {
	ev := llm.SSEEvent{Data: "[DONE]"}
	_, kind := decodeEvent(ev)
	if kind != kindDone {
		t.Errorf("kind = %v, want kindDone", kind)
	}
}

func TestDecodeEvent_EmptyChoicesHeartbeat(t *testing.T) {
	ev := llm.SSEEvent{Data: `{"choices":[]}`}
	_, kind := decodeEvent(ev)
	if kind != kindSkip {
		t.Errorf("kind = %v, want kindSkip", kind)
	}
}

func TestDecodeEvent_EmptyContentSkipped(t *testing.T) {
	ev := llm.SSEEvent{Data: `{"choices":[{"delta":{}}]}`}
	_, kind := decodeEvent(ev)
	if kind != kindSkip {
		t.Errorf("kind = %v, want kindSkip", kind)
	}
}

func TestDecodeEvent_MalformedJSONSkipped(t *testing.T) {
	ev := llm.SSEEvent{Data: `not json`}
	_, kind := decodeEvent(ev)
	if kind != kindSkip {
		t.Errorf("kind = %v, want kindSkip on malformed input", kind)
	}
}

func TestDecodeEvent_NoData(t *testing.T) {
	_, kind := decodeEvent(llm.SSEEvent{})
	if kind != kindSkip {
		t.Errorf("kind = %v, want kindSkip on empty event", kind)
	}
}

func TestDecodeEvent_LengthFinishReasonTruncates(t *testing.T) {
	// The trailing chunk carries finish_reason "length" with empty content:
	// it is skipped for text but must flag truncation.
	ev := llm.SSEEvent{Data: `{"choices":[{"delta":{},"finish_reason":"length"}]}`}
	got, kind := decodeEvent(ev)
	if kind != kindSkip {
		t.Errorf("kind = %v, want kindSkip", kind)
	}
	if !got.Truncated {
		t.Error("finish_reason length should set Truncated")
	}
}

func TestDecodeEvent_StopFinishReasonNotTruncated(t *testing.T) {
	ev := llm.SSEEvent{Data: `{"choices":[{"delta":{},"finish_reason":"stop"}]}`}
	got, _ := decodeEvent(ev)
	if got.Truncated {
		t.Error("finish_reason stop must not set Truncated")
	}
}
