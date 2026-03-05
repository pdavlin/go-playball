package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pdavlin/go-playball/internal/config"
	"github.com/pdavlin/go-playball/internal/ui"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Handle CLI commands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "config":
			handleConfigCommand(cfg)
			return
		case "help", "-h", "--help":
			printHelp()
			return
		case "version", "-v", "--version":
			fmt.Println("go-playball version 1.0.0")
			return
		}
	}

	// Create and start the TUI application
	model := ui.NewModel(cfg)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handleConfigCommand(cfg *config.Config) {
	args := os.Args[2:]

	if len(args) == 0 {
		printConfig(cfg)
		return
	}

	// Handle --unset
	if args[0] == "--unset" {
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: go-playball config --unset <key>")
			os.Exit(1)
		}
		if err := cfg.UnsetKey(args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Reset '%s' to default\n", args[1])
		return
	}

	key := args[0]

	// Get single key value
	if len(args) == 1 {
		val, err := cfg.GetKey(key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "Available keys:\n")
			for _, k := range config.ValidKeys() {
				fmt.Fprintf(os.Stderr, "  %s\n", k)
			}
			os.Exit(1)
		}
		fmt.Println(val)
		return
	}

	// Set key value
	value := strings.Join(args[1:], " ")
	if err := cfg.SetKey(key, value); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if key == "favorite_teams" {
		fmt.Printf("Added '%s' to favorite teams\n", value)
	} else {
		fmt.Printf("Set '%s' to '%s'\n", key, value)
	}
}

func printConfig(cfg *config.Config) {
	fmt.Println("Current Configuration:")
	fmt.Printf("Favorite Teams: %v\n", cfg.FavoriteTeams)
	fmt.Println("Colors:")
	fmt.Printf("  primary:    %s\n", cfg.Colors.Primary)
	fmt.Printf("  secondary:  %s\n", cfg.Colors.Secondary)
	fmt.Printf("  accent:     %s\n", cfg.Colors.Accent)
	fmt.Printf("  error:      %s\n", cfg.Colors.Error)
	fmt.Printf("  success:    %s\n", cfg.Colors.Success)
	fmt.Println("Event Colors:")
	fmt.Printf("  inning_header:  %s\n", cfg.EventColors.InningHeader)
	fmt.Printf("  strikeout:      %s\n", cfg.EventColors.Strikeout)
	fmt.Printf("  walk:           %s\n", cfg.EventColors.Walk)
	fmt.Printf("  in_play_no_out: %s\n", cfg.EventColors.InPlayNoOut)
	fmt.Printf("  in_play_out:    %s\n", cfg.EventColors.InPlayOut)
	fmt.Printf("  default_event:  %s\n", cfg.EventColors.DefaultEvent)
	fmt.Printf("  action_event:   %s\n", cfg.EventColors.ActionEvent)
	fmt.Printf("  scoring_play:   %s\n", cfg.EventColors.ScoringPlay)
	fmt.Printf("  score_badge_fg: %s\n", cfg.EventColors.ScoreBadgeFg)
	fmt.Printf("  score_badge_bg: %s\n", cfg.EventColors.ScoreBadgeBg)
	fmt.Printf("  live_inning:    %s\n", cfg.EventColors.LiveInning)
}

func printHelp() {
	help := `
go-playball - A terminal-based MLB game viewer

USAGE:
    go-playball                            Start the application
    go-playball config                     Show current configuration
    go-playball config <key>               Get a configuration value
    go-playball config <key> <value>       Set a configuration value
    go-playball config --unset <key>       Reset a key to its default
    go-playball help                       Show this help message
    go-playball version                    Show version information

KEYBOARD SHORTCUTS:
    While running:
    c                     Switch to schedule view
    s                     Switch to standings view
    up/down or j/k        Navigate items
    enter                 View selected game
    p                     Previous day (schedule view)
    n                     Next day (schedule view)
    t                     Today (schedule view)
    g                     Scroll to top (game view)
    G                     Scroll to bottom (game view)
    q                     Quit

CONFIGURATION:
    Config file location: ~/.config/go-playball/config.json

    Available config keys:
    favorite_teams              Add a favorite team (use team's full name)
    colors.primary              Primary theme color (hex)
    colors.secondary            Secondary theme color (hex)
    colors.accent               Accent theme color (hex)
    colors.error                Error color (hex)
    colors.success              Success color (hex)
    event_colors.inning_header  Inning header bracket color
    event_colors.strikeout      Strikeout event bracket color
    event_colors.walk           Walk event bracket color
    event_colors.in_play_no_out In-play (no out) bracket color
    event_colors.in_play_out    In-play (out) bracket color
    event_colors.default_event  Default event bracket color
    event_colors.action_event   Action event bracket color
    event_colors.scoring_play   Scoring play description color
    event_colors.score_badge_fg Score badge text color
    event_colors.score_badge_bg Score badge background color
    event_colors.live_inning    Live inning indicator color

    Colors accept hex values (#FF5555) or ANSI indices (1-7).

EXAMPLES:
    go-playball config favorite_teams "New York Yankees"
    go-playball config colors.primary "#00D9FF"
    go-playball config event_colors.walk "#00FF00"
    go-playball config --unset event_colors.walk

For more information, visit: https://github.com/pdavlin/go-playball
`
	fmt.Println(help)
}
