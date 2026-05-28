// Package llm defines the provider-neutral interface for streaming
// text generation. Concrete providers (e.g. internal/llm/anthropic)
// implement Provider. The factory that selects a provider from config
// lives in internal/scouting to avoid a llm -> provider -> llm import
// cycle while keeping the scouting package free of provider-specific
// types.
package llm

import "context"

// Role identifies the speaker of a Message.
type Role string

const (
	RoleSystem Role = "system"
	RoleUser   Role = "user"
)

// Message is one turn of a chat-style request.
type Message struct {
	Role    Role
	Content string
}

// Request is the provider-neutral generation request.
type Request struct {
	Model       string
	Messages    []Message
	MaxTokens   int
	Temperature float64
}

// EventKind discriminates the Event union.
type EventKind int

const (
	// EventDelta carries a chunk of generated text in Event.Text.
	EventDelta EventKind = iota
	// EventDone is the final event in a successful stream. Channel
	// closes immediately after.
	EventDone
	// EventError carries a stream-level error in Event.Err. Channel
	// closes immediately after.
	EventError
)

// Event is one item on a provider's output channel.
type Event struct {
	Kind EventKind
	Text string
	Err  error
}

// Provider is implemented by streaming generation backends.
type Provider interface {
	// Stream returns a channel that yields deltas, an EventDone, or an
	// EventError, then closes. Cancelling ctx must stop the underlying
	// HTTP request and close the channel promptly. Setup errors (bad
	// URL, no API key, etc.) are returned from Stream; once the stream
	// is open, all errors arrive in-band as EventError.
	Stream(ctx context.Context, req Request) (<-chan Event, error)
}
