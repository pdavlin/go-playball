package openaicompat

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

func TestStream_NoAuthHeaderWhenKeyEmpty(t *testing.T) {
	var hadAuth bool
	var gotPath string

	stream := strings.Join([]string{
		`data: {"choices":[{"delta":{"content":"ok"}}]}`,
		``,
		`data: [DONE]`,
		``,
	}, "\n")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, hadAuth = r.Header["Authorization"]
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, stream)
	}))
	defer srv.Close()

	c := New(config.Scouting{
		Provider: "openai-compatible",
		APIKey:   "",
		Model:    "llama3.2",
		BaseURL:  srv.URL,
	})
	ch, err := c.Stream(context.Background(), llm.Request{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	events := collect(t, ch, 2*time.Second)

	if hadAuth {
		t.Error("Authorization header should be absent when api_key is empty")
	}
	if gotPath != "/v1/chat/completions" {
		t.Errorf("path = %q, want /v1/chat/completions", gotPath)
	}
	var text string
	for _, ev := range events {
		if ev.Kind == llm.EventDelta {
			text += ev.Text
		}
	}
	if text != "ok" {
		t.Errorf("text = %q, want %q", text, "ok")
	}
}

func TestStream_BaseURLTrailingSlashJoinedCleanly(t *testing.T) {
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	c := New(config.Scouting{
		BaseURL: srv.URL + "/",
		Model:   "m",
	})
	ch, err := c.Stream(context.Background(), llm.Request{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	collect(t, ch, 2*time.Second)

	if gotPath != "/v1/chat/completions" {
		t.Errorf("path = %q, want single-slash join", gotPath)
	}
}

func TestStream_EOFWithoutDoneCompletes(t *testing.T) {
	// Simulate a server that closes without [DONE].
	stream := strings.Join([]string{
		`data: {"choices":[{"delta":{"content":"abc"}}]}`,
		``,
	}, "\n")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, stream)
	}))
	defer srv.Close()

	c := New(config.Scouting{BaseURL: srv.URL, Model: "m"})
	ch, err := c.Stream(context.Background(), llm.Request{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	events := collect(t, ch, 2*time.Second)
	var sawDone bool
	for _, ev := range events {
		if ev.Kind == llm.EventDone {
			sawDone = true
		}
	}
	if !sawDone {
		t.Error("EOF without [DONE] should synthesize EventDone")
	}
}

func TestStream_EmptyBaseURL(t *testing.T) {
	c := New(config.Scouting{BaseURL: "", Model: "m"})
	_, err := c.Stream(context.Background(), llm.Request{})
	if err == nil {
		t.Fatal("expected error for empty base_url")
	}
}
