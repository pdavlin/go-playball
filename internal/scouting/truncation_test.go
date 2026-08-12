package scouting

import (
	"context"
	"strings"
	"testing"

	"github.com/pdavlin/go-playball/internal/config"
	"github.com/pdavlin/go-playball/internal/llm"
)

// drainOrchestration feeds evs through runOrchestration against a temp-dir
// cache and returns the concatenated rendered text plus the cache handle.
func drainOrchestration(t *testing.T, evs []llm.Event) (string, *Cache) {
	t.Helper()
	cache := NewCache(t.TempDir())
	stream := make(chan llm.Event, len(evs))
	for _, ev := range evs {
		stream <- ev
	}
	close(stream)

	out := make(chan Event, 32)
	go runOrchestration(context.Background(), stream, out, cache,
		config.Scouting{Provider: "anthropic", Model: "m"}, "hash", "fp", 42)

	var rendered string
	for ev := range out {
		if ev.Kind == EventDelta {
			rendered += ev.Text
		}
	}
	return rendered, cache
}

func TestRunOrchestration_TruncatedNotCached(t *testing.T) {
	rendered, cache := drainOrchestration(t, []llm.Event{
		{Kind: llm.EventDelta, Text: "partial report"},
		{Kind: llm.EventDone, Truncated: true},
	})

	entry, _ := cache.Load(42)
	if entry != nil {
		t.Errorf("truncated response must not be cached, got %+v", entry)
	}
	if !strings.Contains(rendered, truncationMarker) {
		t.Errorf("rendered text should include the truncation marker, got %q", rendered)
	}
}

func TestRunOrchestration_CompleteIsCached(t *testing.T) {
	rendered, cache := drainOrchestration(t, []llm.Event{
		{Kind: llm.EventDelta, Text: "full report"},
		{Kind: llm.EventDone},
	})

	entry, err := cache.Load(42)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if entry == nil || entry.Body != "full report" {
		t.Errorf("complete response should be cached with full body, got %+v", entry)
	}
	if strings.Contains(rendered, truncationMarker) {
		t.Errorf("complete response should not carry the truncation marker, got %q", rendered)
	}
}
