package openrouter

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pdavlin/go-playball/internal/config"
	"github.com/pdavlin/go-playball/internal/llm"
)

func newTestClient(srv *httptest.Server) *Client {
	return New(config.Scouting{
		Provider: "openrouter",
		APIKey:   "test-key",
		Model:    "anthropic/claude-3.5-haiku",
		BaseURL:  srv.URL,
	})
}

func collect(t *testing.T, ch <-chan llm.Event, timeout time.Duration) []llm.Event {
	t.Helper()
	var events []llm.Event
	deadline := time.After(timeout)
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return events
			}
			events = append(events, ev)
		case <-deadline:
			t.Fatalf("timed out collecting events, got %d so far", len(events))
		}
	}
}

func TestStream_HeadersAndDeltas(t *testing.T) {
	var gotAuth, gotReferer, gotTitle string

	stream := strings.Join([]string{
		`data: {"choices":[{"delta":{"content":"Hel"}}]}`,
		``,
		`data: {"choices":[{"delta":{"content":"lo"}}]}`,
		``,
		`data: [DONE]`,
		``,
	}, "\n")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotReferer = r.Header.Get("HTTP-Referer")
		gotTitle = r.Header.Get("X-Title")
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
	events := collect(t, ch, 2*time.Second)

	if gotAuth != "Bearer test-key" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if gotReferer != httpReferer {
		t.Errorf("HTTP-Referer = %q", gotReferer)
	}
	if gotTitle != xTitle {
		t.Errorf("X-Title = %q", gotTitle)
	}

	var text string
	var sawDone bool
	for _, ev := range events {
		switch ev.Kind {
		case llm.EventDelta:
			text += ev.Text
		case llm.EventDone:
			sawDone = true
		case llm.EventError:
			t.Fatalf("unexpected error: %v", ev.Err)
		}
	}
	if text != "Hello" {
		t.Errorf("text = %q, want %q", text, "Hello")
	}
	if !sawDone {
		t.Error("missing EventDone")
	}
}

func TestStream_401YieldsErrorEvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":{"message":"invalid key"}}`)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	ch, err := c.Stream(context.Background(), llm.Request{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	events := collect(t, ch, 2*time.Second)
	if len(events) == 0 || events[0].Kind != llm.EventError {
		t.Fatalf("expected EventError, got %+v", events)
	}
	if !strings.Contains(events[0].Err.Error(), "401") {
		t.Errorf("error should mention 401: %v", events[0].Err)
	}
}

func TestStream_429AnnotatedWithSuggestion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"error":{"message":"rate limited"}}`)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	ch, err := c.Stream(context.Background(), llm.Request{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	events := collect(t, ch, 2*time.Second)
	if len(events) == 0 || events[0].Kind != llm.EventError {
		t.Fatalf("expected EventError, got %+v", events)
	}
	if !strings.Contains(events[0].Err.Error(), "try again") {
		t.Errorf("429 should include retry hint: %v", events[0].Err)
	}
}

func TestStream_EmptyAPIKey(t *testing.T) {
	c := New(config.Scouting{Provider: "openrouter", APIKey: "", Model: "m"})
	_, err := c.Stream(context.Background(), llm.Request{})
	if err == nil {
		t.Fatal("expected error for empty api key")
	}
}
