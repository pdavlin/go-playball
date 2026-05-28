package oai

import (
	"encoding/json"

	"github.com/pdavlin/go-playball/internal/llm"
)

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Stream      bool          `json:"stream"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
}

// buildBody serializes req into an OpenAI chat-completion request body.
// Unlike Anthropic, OpenAI puts the system prompt in messages[0] with
// role=system rather than in a top-level field.
func buildBody(req llm.Request) ([]byte, error) {
	msgs := make([]chatMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		role := "user"
		switch m.Role {
		case llm.RoleSystem:
			role = "system"
		case llm.RoleUser:
			role = "user"
		}
		msgs = append(msgs, chatMessage{Role: role, Content: m.Content})
	}
	body := chatRequest{
		Model:       req.Model,
		Messages:    msgs,
		Stream:      true,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}
	return json.Marshal(body)
}
