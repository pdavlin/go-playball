package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pdavlin/go-playball/internal/config"
	"github.com/pdavlin/go-playball/internal/llm"
	"github.com/pdavlin/go-playball/internal/scouting"
)

const scoutingTestTimeout = 30 * time.Second

// handleScoutingCommand dispatches `go-playball scouting <sub>`. Currently
// the only subcommand is `test`, which pings the configured provider with a
// tiny prompt to validate credentials and connectivity without involving
// the TUI or cache.
func handleScoutingCommand(cfg *config.Config) {
	args := os.Args[2:]
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: go-playball scouting test")
		os.Exit(1)
	}
	switch args[0] {
	case "test":
		runScoutingTest(cfg)
	default:
		fmt.Fprintf(os.Stderr, "Unknown scouting subcommand: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "Usage: go-playball scouting test")
		os.Exit(1)
	}
}

func runScoutingTest(cfg *config.Config) {
	if !cfg.ScoutingEnabled() {
		fmt.Fprintln(os.Stderr, "Scouting is not configured.")
		fmt.Fprintln(os.Stderr, "Set scouting.provider, scouting.api_key, and scouting.model first.")
		fmt.Fprintln(os.Stderr, "Example:")
		fmt.Fprintln(os.Stderr, "  go-playball config scouting.provider anthropic")
		fmt.Fprintln(os.Stderr, "  go-playball config scouting.api_key sk-ant-...")
		fmt.Fprintln(os.Stderr, "  go-playball config scouting.model claude-haiku-4-5-20251001")
		os.Exit(2)
	}

	scfg := cfg.ScoutingValue()
	provider, err := scouting.NewProvider(scfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Provider error: %v\n", err)
		os.Exit(3)
	}

	fmt.Printf("provider: %s\n", scfg.Provider)
	fmt.Printf("model:    %s\n", scfg.Model)

	ctx, cancel := context.WithTimeout(context.Background(), scoutingTestTimeout)
	defer cancel()

	stream, err := provider.Stream(ctx, llm.Request{
		Model: scfg.Model,
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: "Respond with the single word: ok"},
		},
		MaxTokens:   16,
		Temperature: 0,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Stream setup failed: %v\n", err)
		os.Exit(3)
	}

	for ev := range stream {
		switch ev.Kind {
		case llm.EventDelta:
			fmt.Print(ev.Text)
		case llm.EventDone:
			fmt.Println()
		case llm.EventError:
			fmt.Println()
			fmt.Fprintf(os.Stderr, "Stream error: %v\n", ev.Err)
			os.Exit(1)
		}
	}
}
