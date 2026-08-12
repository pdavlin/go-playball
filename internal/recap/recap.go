package recap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"time"

	"github.com/pdavlin/go-playball/internal/api"
	"github.com/pdavlin/go-playball/internal/config"
	"github.com/pdavlin/go-playball/internal/llm"
	"github.com/pdavlin/go-playball/internal/llm/anthropic"
	"github.com/pdavlin/go-playball/internal/llm/openaicompat"
	"github.com/pdavlin/go-playball/internal/llm/openrouter"
	"github.com/pdavlin/go-playball/internal/reportcache"
)

const cacheSchemaVersion = 1

// truncationMarker is appended to the streamed body when the provider stopped
// at the output token cap, so the cutoff is visible rather than silent.
const truncationMarker = " …(truncated: hit max_tokens)"

// EventKind mirrors scouting.EventKind so the modal can drain both
// streams through a small adapter without depending on either package.
type EventKind int

const (
	EventDelta EventKind = iota
	EventDone
	EventError
)

// Event is the stream payload emitted by Generate.
type Event struct {
	Kind     EventKind
	Text     string
	Err      error
	Cached   bool
	CachedAt time.Time
}

// NewCache returns a recap cache rooted at dir.
func NewCache(dir string) *reportcache.Cache {
	return reportcache.NewCache(dir, cacheSchemaVersion)
}

// DefaultCacheDir returns the platform-appropriate cache directory used
// by the go-playball binary for recap reports.
func DefaultCacheDir() (string, error) {
	return reportcache.DefaultDir("recap")
}

// Generate builds a recap for g and streams it. Returns ErrNotFinal or
// ErrIncompletePayload (wrapped from BuildContext) without making an LLM
// call when the game cannot be recapped.
func Generate(
	ctx context.Context,
	cfg config.Scouting,
	cache *reportcache.Cache,
	c *api.Client,
	g *api.Game,
) (<-chan Event, error) {
	provider, err := newProvider(cfg)
	if err != nil {
		return nil, err
	}

	rctx, err := BuildContext(ctx, c, g)
	if err != nil {
		return nil, err
	}
	system, user := RenderPrompt(rctx)
	hash := promptHash(system, user, cfg.Model)

	out := make(chan Event, 16)

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

	go runOrchestration(ctx, stream, out, cache, cfg, hash, g.ID)
	return out, nil
}

func runOrchestration(
	ctx context.Context,
	stream <-chan llm.Event,
	out chan<- Event,
	cache *reportcache.Cache,
	cfg config.Scouting,
	hash string,
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
		_ = cache.Save(reportcache.CacheEntry{
			GamePk:      gamePk,
			Provider:    cfg.Provider,
			Model:       cfg.Model,
			PromptHash:  hash,
			GeneratedAt: time.Now().UTC(),
			Body:        body,
		})
	}
}

// Delete removes any cached recap for gamePk.
func Delete(cache *reportcache.Cache, gamePk int) error {
	if cache == nil {
		return nil
	}
	return cache.Delete(gamePk)
}

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

func promptHash(system, user, model string) string {
	h := sha256.New()
	h.Write([]byte(system))
	h.Write([]byte{0})
	h.Write([]byte(user))
	h.Write([]byte{0})
	h.Write([]byte(model))
	sum := h.Sum(nil)
	return hex.EncodeToString(sum[:8])
}
