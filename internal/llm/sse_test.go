package llm

import (
	"strings"
	"testing"
)

func TestScanSSE_BasicEvents(t *testing.T) {
	input := "event: a\ndata: 1\n\nevent: b\ndata: 2\n\n"
	var got []SSEEvent
	if err := ScanSSE(strings.NewReader(input), func(ev SSEEvent) bool {
		got = append(got, ev)
		return true
	}); err != nil {
		t.Fatalf("ScanSSE: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 events, got %d: %+v", len(got), got)
	}
	if got[0].Type != "a" || got[0].Data != "1" {
		t.Errorf("event 0: %+v", got[0])
	}
	if got[1].Type != "b" || got[1].Data != "2" {
		t.Errorf("event 1: %+v", got[1])
	}
}

func TestScanSSE_MultilineData(t *testing.T) {
	input := "data: line1\ndata: line2\n\n"
	var got SSEEvent
	_ = ScanSSE(strings.NewReader(input), func(ev SSEEvent) bool {
		got = ev
		return false
	})
	if got.Data != "line1\nline2" {
		t.Errorf("want joined data, got %q", got.Data)
	}
}

func TestScanSSE_CRLF(t *testing.T) {
	input := "event: x\r\ndata: y\r\n\r\n"
	var got SSEEvent
	_ = ScanSSE(strings.NewReader(input), func(ev SSEEvent) bool {
		got = ev
		return false
	})
	if got.Type != "x" || got.Data != "y" {
		t.Errorf("got %+v", got)
	}
}

func TestScanSSE_IgnoresComments(t *testing.T) {
	input := ": keepalive\ndata: 1\n\n"
	count := 0
	_ = ScanSSE(strings.NewReader(input), func(ev SSEEvent) bool {
		count++
		if ev.Data != "1" {
			t.Errorf("got %q", ev.Data)
		}
		return true
	})
	if count != 1 {
		t.Errorf("want 1 event, got %d", count)
	}
}

func TestScanSSE_TrailingEventWithoutBlankLine(t *testing.T) {
	input := "data: tail"
	var got SSEEvent
	_ = ScanSSE(strings.NewReader(input), func(ev SSEEvent) bool {
		got = ev
		return true
	})
	if got.Data != "tail" {
		t.Errorf("flush failed, got %q", got.Data)
	}
}
