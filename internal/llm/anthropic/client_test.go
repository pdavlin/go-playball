package anthropic

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
	c := New(config.Scouting{
		Provider: "anthropic",
		APIKey:   "test-key",
		Model:    "claude-test",
		BaseURL:  srv.URL,
	})
	return c
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

func TestStream_Success(t *testing.T) {
	stream := strings.Join([]string{
		`event: message_start`,
		`data: {"type":"message_start"}`,
		``,
		`event: content_block_delta`,
		`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"Hello"}}`,
		``,
		`event: content_block_delta`,
		`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":" world"}}`,
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
	events := collect(t, ch, 2*time.Second)
	var text string
	var sawDone bool
	for _, ev := range events {
		switch ev.Kind {
		case llm.EventDelta:
			text += ev.Text
		case llm.EventDone:
			sawDone = true
		case llm.EventError:
			t.Fatalf("unexpected error event: %v", ev.Err)
		}
	}
	if text != "Hello world" {
		t.Errorf("got text %q", text)
	}
	if !sawDone {
		t.Error("missing EventDone")
	}
}

func TestStream_Non2xxYieldsErrorEvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":{"type":"authentication_error","message":"invalid key"}}`)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	ch, err := c.Stream(context.Background(), llm.Request{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "x"}},
	})
	if err != nil {
		t.Fatalf("setup err: %v", err)
	}
	events := collect(t, ch, time.Second)
	if len(events) != 1 || events[0].Kind != llm.EventError {
		t.Fatalf("want one error event, got %+v", events)
	}
	if !strings.Contains(events[0].Err.Error(), "401") {
		t.Errorf("expected status in error, got %v", events[0].Err)
	}
}

func TestStream_ContextCancelCloses(t *testing.T) {
	// Server sends one delta then blocks; client cancels context.
	releaseCancel := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		fmt.Fprint(w, "event: content_block_delta\n")
		fmt.Fprint(w, `data: {"type":"content_block_delta","delta":{"text":"x"}}`+"\n\n")
		if flusher != nil {
			flusher.Flush()
		}
		<-r.Context().Done()
		close(releaseCancel)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := c.Stream(ctx, llm.Request{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "x"}},
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Drain first event.
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("no first event")
	}

	cancel()
	// Channel must close within 100ms.
	closed := make(chan struct{})
	go func() {
		for range ch {
		}
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("channel did not close after cancel")
	}
}
