package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds application configuration
type Config struct {
	FavoriteTeams []string         `json:"favorite_teams"`
	Colors        ColorConfig      `json:"colors"`
	EventColors   EventColorConfig `json:"event_colors"`
}

// ColorConfig holds color customization
type ColorConfig struct {
	Primary   string `json:"primary"`
	Secondary string `json:"secondary"`
	Accent    string `json:"accent"`
	Error     string `json:"error"`
	Success   string `json:"success"`
}

// EventColorConfig holds event-specific color customization
type EventColorConfig struct {
	InningHeader string `json:"inning_header"`
	Strikeout    string `json:"strikeout"`
	Walk         string `json:"walk"`
	InPlayNoOut  string `json:"in_play_no_out"`
	InPlayOut    string `json:"in_play_out"`
	DefaultEvent string `json:"default_event"`
	ActionEvent  string `json:"action_event"`
	ScoringPlay  string `json:"scoring_play"`
	ScoreBadgeFg string `json:"score_badge_fg"`
	ScoreBadgeBg string `json:"score_badge_bg"`
	LiveInning   string `json:"live_inning"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		FavoriteTeams: []string{},
		Colors: ColorConfig{
			Primary:   "#00D9FF",
			Secondary: "#FFB86C",
			Accent:    "#50FA7B",
			Error:     "#FF5555",
			Success:   "#50FA7B",
		},
		EventColors: EventColorConfig{
			InningHeader: "7",
			Strikeout:    "1",
			Walk:         "2",
			InPlayNoOut:  "4",
			InPlayOut:    "7",
			DefaultEvent: "8",
			ActionEvent:  "8",
			ScoringPlay:  "#FF6B6B",
			ScoreBadgeFg: "#F8F8F2",
			ScoreBadgeBg: "#44475A",
			LiveInning:   "#FF6B6B",
		},
	}
}

// Load reads configuration from disk, or creates default if not found
func Load() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	// If config doesn't exist, create default
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := DefaultConfig()
		if err := cfg.Save(); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	cfg.fillDefaults()

	return &cfg, nil
}

// fillDefaults fills empty fields with default values.
// Handles migration from older config files missing new fields.
func (c *Config) fillDefaults() {
	defaults := DefaultConfig()

	if c.Colors.Primary == "" {
		c.Colors.Primary = defaults.Colors.Primary
	}
	if c.Colors.Secondary == "" {
		c.Colors.Secondary = defaults.Colors.Secondary
	}
	if c.Colors.Accent == "" {
		c.Colors.Accent = defaults.Colors.Accent
	}
	if c.Colors.Error == "" {
		c.Colors.Error = defaults.Colors.Error
	}
	if c.Colors.Success == "" {
		c.Colors.Success = defaults.Colors.Success
	}

	if c.EventColors.InningHeader == "" {
		c.EventColors.InningHeader = defaults.EventColors.InningHeader
	}
	if c.EventColors.Strikeout == "" {
		c.EventColors.Strikeout = defaults.EventColors.Strikeout
	}
	if c.EventColors.Walk == "" {
		c.EventColors.Walk = defaults.EventColors.Walk
	}
	if c.EventColors.InPlayNoOut == "" {
		c.EventColors.InPlayNoOut = defaults.EventColors.InPlayNoOut
	}
	if c.EventColors.InPlayOut == "" {
		c.EventColors.InPlayOut = defaults.EventColors.InPlayOut
	}
	if c.EventColors.DefaultEvent == "" {
		c.EventColors.DefaultEvent = defaults.EventColors.DefaultEvent
	}
	if c.EventColors.ActionEvent == "" {
		c.EventColors.ActionEvent = defaults.EventColors.ActionEvent
	}
	if c.EventColors.ScoringPlay == "" {
		c.EventColors.ScoringPlay = defaults.EventColors.ScoringPlay
	}
	if c.EventColors.ScoreBadgeFg == "" {
		c.EventColors.ScoreBadgeFg = defaults.EventColors.ScoreBadgeFg
	}
	if c.EventColors.ScoreBadgeBg == "" {
		c.EventColors.ScoreBadgeBg = defaults.EventColors.ScoreBadgeBg
	}
	if c.EventColors.LiveInning == "" {
		c.EventColors.LiveInning = defaults.EventColors.LiveInning
	}
}

// Save writes configuration to disk
func (c *Config) Save() error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// getConfigPath returns the path to the config file
func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, ".config", "go-playball", "config.json"), nil
}

// IsFavoriteTeam checks if a team name is in favorites
func (c *Config) IsFavoriteTeam(teamName string) bool {
	for _, fav := range c.FavoriteTeams {
		if fav == teamName {
			return true
		}
	}
	return false
}

// GetKey returns the current value for a dot-notated config key
func (c *Config) GetKey(key string) (string, error) {
	switch key {
	case "favorite_teams":
		return strings.Join(c.FavoriteTeams, ", "), nil
	case "colors.primary":
		return c.Colors.Primary, nil
	case "colors.secondary":
		return c.Colors.Secondary, nil
	case "colors.accent":
		return c.Colors.Accent, nil
	case "colors.error":
		return c.Colors.Error, nil
	case "colors.success":
		return c.Colors.Success, nil
	case "event_colors.inning_header":
		return c.EventColors.InningHeader, nil
	case "event_colors.strikeout":
		return c.EventColors.Strikeout, nil
	case "event_colors.walk":
		return c.EventColors.Walk, nil
	case "event_colors.in_play_no_out":
		return c.EventColors.InPlayNoOut, nil
	case "event_colors.in_play_out":
		return c.EventColors.InPlayOut, nil
	case "event_colors.default_event":
		return c.EventColors.DefaultEvent, nil
	case "event_colors.action_event":
		return c.EventColors.ActionEvent, nil
	case "event_colors.scoring_play":
		return c.EventColors.ScoringPlay, nil
	case "event_colors.score_badge_fg":
		return c.EventColors.ScoreBadgeFg, nil
	case "event_colors.score_badge_bg":
		return c.EventColors.ScoreBadgeBg, nil
	case "event_colors.live_inning":
		return c.EventColors.LiveInning, nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}

// SetKey sets a config value by dot-notated key and saves to disk
func (c *Config) SetKey(key, value string) error {
	switch key {
	case "favorite_teams":
		for _, team := range c.FavoriteTeams {
			if team == value {
				return nil
			}
		}
		c.FavoriteTeams = append(c.FavoriteTeams, value)
	case "colors.primary":
		c.Colors.Primary = value
	case "colors.secondary":
		c.Colors.Secondary = value
	case "colors.accent":
		c.Colors.Accent = value
	case "colors.error":
		c.Colors.Error = value
	case "colors.success":
		c.Colors.Success = value
	case "event_colors.inning_header":
		c.EventColors.InningHeader = value
	case "event_colors.strikeout":
		c.EventColors.Strikeout = value
	case "event_colors.walk":
		c.EventColors.Walk = value
	case "event_colors.in_play_no_out":
		c.EventColors.InPlayNoOut = value
	case "event_colors.in_play_out":
		c.EventColors.InPlayOut = value
	case "event_colors.default_event":
		c.EventColors.DefaultEvent = value
	case "event_colors.action_event":
		c.EventColors.ActionEvent = value
	case "event_colors.scoring_play":
		c.EventColors.ScoringPlay = value
	case "event_colors.score_badge_fg":
		c.EventColors.ScoreBadgeFg = value
	case "event_colors.score_badge_bg":
		c.EventColors.ScoreBadgeBg = value
	case "event_colors.live_inning":
		c.EventColors.LiveInning = value
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return c.Save()
}

// UnsetKey resets a config key to its default value and saves to disk
func (c *Config) UnsetKey(key string) error {
	defaults := DefaultConfig()
	switch key {
	case "favorite_teams":
		c.FavoriteTeams = []string{}
	case "colors.primary":
		c.Colors.Primary = defaults.Colors.Primary
	case "colors.secondary":
		c.Colors.Secondary = defaults.Colors.Secondary
	case "colors.accent":
		c.Colors.Accent = defaults.Colors.Accent
	case "colors.error":
		c.Colors.Error = defaults.Colors.Error
	case "colors.success":
		c.Colors.Success = defaults.Colors.Success
	case "event_colors.inning_header":
		c.EventColors.InningHeader = defaults.EventColors.InningHeader
	case "event_colors.strikeout":
		c.EventColors.Strikeout = defaults.EventColors.Strikeout
	case "event_colors.walk":
		c.EventColors.Walk = defaults.EventColors.Walk
	case "event_colors.in_play_no_out":
		c.EventColors.InPlayNoOut = defaults.EventColors.InPlayNoOut
	case "event_colors.in_play_out":
		c.EventColors.InPlayOut = defaults.EventColors.InPlayOut
	case "event_colors.default_event":
		c.EventColors.DefaultEvent = defaults.EventColors.DefaultEvent
	case "event_colors.action_event":
		c.EventColors.ActionEvent = defaults.EventColors.ActionEvent
	case "event_colors.scoring_play":
		c.EventColors.ScoringPlay = defaults.EventColors.ScoringPlay
	case "event_colors.score_badge_fg":
		c.EventColors.ScoreBadgeFg = defaults.EventColors.ScoreBadgeFg
	case "event_colors.score_badge_bg":
		c.EventColors.ScoreBadgeBg = defaults.EventColors.ScoreBadgeBg
	case "event_colors.live_inning":
		c.EventColors.LiveInning = defaults.EventColors.LiveInning
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return c.Save()
}

// ValidKeys returns all valid config key names
func ValidKeys() []string {
	return []string{
		"favorite_teams",
		"colors.primary",
		"colors.secondary",
		"colors.accent",
		"colors.error",
		"colors.success",
		"event_colors.inning_header",
		"event_colors.strikeout",
		"event_colors.walk",
		"event_colors.in_play_no_out",
		"event_colors.in_play_out",
		"event_colors.default_event",
		"event_colors.action_event",
		"event_colors.scoring_play",
		"event_colors.score_badge_fg",
		"event_colors.score_badge_bg",
		"event_colors.live_inning",
	}
}
