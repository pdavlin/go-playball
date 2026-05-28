// Package openrouter implements an llm.Provider backed by OpenRouter's
// OpenAI-compatible chat-completion API. The wire format is shared with
// other OpenAI-compatible servers, so this file is a thin shim over
// internal/llm/oai.
package openrouter

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pdavlin/go-playball/internal/config"
	"github.com/pdavlin/go-playball/internal/llm"
	"github.com/pdavlin/go-playball/internal/llm/oai"
)

const (
	defaultEndpoint = "https://openrouter.ai/api/v1/chat/completions"
	httpReferer     = "https://github.com/pdavlin/go-playball"
	xTitle          = "go-playball"
)

// Client is an OpenRouter-backed llm.Provider.
type Client struct {
	apiKey      string
	model       string
	endpoint    string
	maxTokens   int
	temperature float64
	http        *http.Client
}

// New builds a Client from a config.Scouting. BaseURL overrides the default
// OpenRouter endpoint when set, primarily for testing.
func New(cfg config.Scouting) *Client {
	endpoint := defaultEndpoint
	if cfg.BaseURL != "" {
		endpoint = cfg.BaseURL
	}
	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1024
	}
	temperature := cfg.Temperature
	if temperature == 0 {
		temperature = 0.4
	}
	return &Client{
		apiKey:      cfg.APIKey,
		model:       cfg.Model,
		endpoint:    endpoint,
		maxTokens:   maxTokens,
		temperature: temperature,
		http:        &http.Client{},
	}
}

// Stream implements llm.Provider.
func (c *Client) Stream(ctx context.Context, req llm.Request) (<-chan llm.Event, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("openrouter: api key is empty")
	}
	if req.Model == "" {
		req.Model = c.model
	}
	if req.MaxTokens <= 0 {
		req.MaxTokens = c.maxTokens
	}
	if req.Temperature == 0 {
		req.Temperature = c.temperature
	}
	headers := map[string]string{
		"Authorization": "Bearer " + c.apiKey,
		"HTTP-Referer":  httpReferer,
		"X-Title":       xTitle,
	}
	return oai.StreamChat(ctx, c.http, c.endpoint, headers, req, "openrouter")
}
