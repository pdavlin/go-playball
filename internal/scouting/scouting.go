package scouting

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"net/url"

	"github.com/pdavlin/go-playball/internal/api"
	"github.com/pdavlin/go-playball/internal/config"
	"github.com/pdavlin/go-playball/internal/llm"
	"github.com/pdavlin/go-playball/internal/llm/anthropic"
	"github.com/pdavlin/go-playball/internal/llm/openaicompat"
	"github.com/pdavlin/go-playball/internal/llm/openrouter"
)

// EventKind is the same disambiguation as llm.EventKind but re-exported so
// callers don't need to import internal/llm.
type EventKind int

const (
	EventDelta EventKind = iota
	EventDone
	EventError
)

// truncationMarker is appended to the streamed body when the provider stopped
// at the output token cap, so the cutoff is visible rather than silent.
const truncationMarker = " …(truncated: hit max_tokens)"

// Event is what scouting.Generate emits. CachedAt is the generated_at of
// the entry when Kind == EventDelta from a cache hit; zero otherwise.
type Event struct {
	Kind     EventKind
	Text     string
	Err      error
	Cached   bool
	CachedAt time.Time
}

// Generate builds a report for g, returning a channel that emits the body
// in pieces and closes on EventDone or EventError. A cache hit produces a
// single EventDelta with the entire body, then EventDone. Cancelling ctx
// stops the underlying LLM stream and closes the channel.
func Generate(ctx context.Context, cfg config.Scouting, cache *Cache, c *api.Client, g *api.Game) (<-chan Event, error) {
	provider, err := newProvider(cfg)
	if err != nil {
		return nil, err
	}

	gctx, err := BuildContext(ctx, c, g)
	if err != nil {
		return nil, err
	}
	system, user := RenderPrompt(gctx)
	fingerprint := lineupFingerprint(gctx.Lineups[0], gctx.Lineups[1])
	hash := promptHash(system, user, cfg.Model, fingerprint)

	out := make(chan Event, 16)

	// Cache hit: serve whole body, no LLM call.
	if cache != nil {
		if entry, _ := cache.Load(g.ID); entry != nil && entry.PromptHash == hash {
			go func() {
				defer close(out)
				out <- Event{
					Kind:     EventDelta,
					Text:     entry.Body,
					Cached:   true,
					CachedAt: entry.GeneratedAt,
				}
				out <- Event{Kind: EventDone, Cached: true, CachedAt: entry.GeneratedAt}
			}()
			return out, nil
		}
	}

	req := llm.Request{
		Model: cfg.Model,
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: system},
			{Role: llm.RoleUser, Content: user},
		},
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
	}

	stream, err := provider.Stream(ctx, req)
	if err != nil {
		return nil, err
	}

	go runOrchestration(ctx, stream, out, cache, cfg, hash, fingerprint, g.ID)
	return out, nil
}

func runOrchestration(
	ctx context.Context,
	stream <-chan llm.Event,
	out chan<- Event,
	cache *Cache,
	cfg config.Scouting,
	promptHash string,
	lineupFingerprint string,
	gamePk int,
) {
	defer close(out)
	var body string
	var sawError bool
	var truncated bool

	for ev := range stream {
		switch ev.Kind {
		case llm.EventDelta:
			body += ev.Text
			select {
			case <-ctx.Done():
				return
			case out <- Event{Kind: EventDelta, Text: ev.Text}:
			}
		case llm.EventDone:
			// A response cut off at the token cap must stay out of the
			// cache and be flagged to the reader. Emit the marker as a
			// delta so the existing renderer shows it without any UI change.
			if ev.Truncated {
				truncated = true
				select {
				case <-ctx.Done():
					return
				case out <- Event{Kind: EventDelta, Text: truncationMarker}:
				}
			}
			select {
			case <-ctx.Done():
				return
			case out <- Event{Kind: EventDone}:
			}
		case llm.EventError:
			sawError = true
			select {
			case <-ctx.Done():
				return
			case out <- Event{Kind: EventError, Err: ev.Err}:
			}
		}
	}

	if !sawError && !truncated && cache != nil && body != "" {
		_ = cache.Save(CacheEntry{
			GamePk:            gamePk,
			Provider:          cfg.Provider,
			Model:             cfg.Model,
			PromptHash:        promptHash,
			LineupFingerprint: lineupFingerprint,
			GeneratedAt:       time.Now().UTC(),
			Body:              body,
		})
	}
}

// Delete removes any cached report for gamePk.
func Delete(cache *Cache, gamePk int) error {
	if cache == nil {
		return nil
	}
	return cache.Delete(gamePk)
}

// NewProvider builds an llm.Provider from cfg. Exported for the
// `scouting test` CLI so it can exercise the same factory the report
// pipeline uses.
func NewProvider(cfg config.Scouting) (llm.Provider, error) {
	return newProvider(cfg)
}

// newProvider builds an llm.Provider from cfg. Kept here (rather than in
// the llm package) to avoid an import cycle: llm/anthropic depends on llm.
func newProvider(cfg config.Scouting) (llm.Provider, error) {
	switch cfg.Provider {
	case "anthropic":
		return anthropic.New(cfg), nil
	case "openrouter":
		return openrouter.New(cfg), nil
	case "openai-compatible":
		if cfg.BaseURL == "" {
			return nil, fmt.Errorf(`scouting.base_url is required when provider="openai-compatible"`)
		}
		u, err := url.Parse(cfg.BaseURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return nil, fmt.Errorf("invalid scouting.base_url %q (must be http or https URL)", cfg.BaseURL)
		}
		return openaicompat.New(cfg), nil
	case "":
		return nil, fmt.Errorf("no scouting provider configured")
	default:
		return nil, fmt.Errorf("unknown scouting provider %q (valid: anthropic, openrouter, openai-compatible)", cfg.Provider)
	}
}

func promptHash(system, user, model, fingerprint string) string {
	h := sha256.New()
	h.Write([]byte(system))
	h.Write([]byte{0})
	h.Write([]byte(user))
	h.Write([]byte{0})
	h.Write([]byte(model))
	h.Write([]byte{0})
	h.Write([]byte(fingerprint))
	sum := h.Sum(nil)
	return hex.EncodeToString(sum[:8])
}
