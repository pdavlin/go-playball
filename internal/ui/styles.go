package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/pdavlin/go-playball/internal/config"
)

var (
	// Base colors
	colorPrimary   = lipgloss.Color("#00D9FF")
	colorSecondary = lipgloss.Color("#FFB86C")
	colorAccent    = lipgloss.Color("#50FA7B")
	colorError     = lipgloss.Color("#FF5555")
	colorSuccess   = lipgloss.Color("#50FA7B")
	colorSubtle    = lipgloss.Color("#6272A4")
	colorBright    = lipgloss.Color("#F8F8F2")

	// Event colors (used in game.go renderAllPlays)
	colorInningHeader = lipgloss.Color("7")
	colorStrikeout    = lipgloss.Color("1")
	colorWalk         = lipgloss.Color("2")
	colorInPlayNoOut  = lipgloss.Color("4")
	colorInPlayOut    = lipgloss.Color("7")
	colorDefaultEvent = lipgloss.Color("8")
	colorActionEvent  = lipgloss.Color("8")
	colorScoringPlay  = lipgloss.Color("#FF6B6B")
	colorScoreBadgeFg = lipgloss.Color("#F8F8F2")
	colorScoreBadgeBg = lipgloss.Color("#44475A")
	colorLiveInning   = lipgloss.Color("#FF6B6B")

	// Title styles with gradient
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			MarginBottom(1)

	// Gradient title helper
	gradientTitleStyle = lipgloss.NewStyle().
				Bold(true).
				MarginBottom(1)

	// Header styles
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorSecondary).
			Padding(0, 1)

	// Selected item style
	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			Reverse(true)

	// Normal item style
	itemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#F8F8F2"})

	// Favorite team style (with star)
	favoriteStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	// Status styles
	liveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorError)

	previewStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#6272A4"})

	finalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#F8F8F2"})

	// Help bar style
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#6272A4"}).
			Padding(0, 1)

	// Error style
	errorStyle = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	// Box style for game details
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 2)

	// Score style
	scoreStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#F8F8F2"})

	// Inning style
	inningStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	// Count indicators - balls (green), strikes/outs (red)
	countFilledStyle = lipgloss.NewStyle().
				Foreground(colorAccent)

	countEmptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#AAAAAA", Dark: "#6272A4"})

	strikeFilledStyle = lipgloss.NewStyle().
				Foreground(colorError)

	strikeEmptyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#DDAAAA", Dark: "#663333"})

	outFilledStyle = lipgloss.NewStyle().
			Foreground(colorError)

	outEmptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#DDAAAA", Dark: "#663333"})

	ballEmptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#AADDAA", Dark: "#336633"})

	// Base runner indicators
	baseFilledStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	baseEmptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#AAAAAA", Dark: "#6272A4"})

	// Table header
	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorSecondary).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(lipgloss.AdaptiveColor{Light: "#CCCCCC", Dark: "#6272A4"})

	// Table cell
	tableCellStyle = lipgloss.NewStyle().
			Padding(0, 1)

	// Strike zone styles
	zoneBorderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#999999", Dark: "#6272A4"})

	zoneEmptyStyle = lipgloss.NewStyle()

	zoneBallStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	zoneBallDimStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#4A8C4A", Dark: "#336633"})

	zoneStrikeStyle = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	zoneStrikeDimStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#8C4A4A", Dark: "#663333"})

	zoneInPlayStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	zoneInPlayDimStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#8C7A3D", Dark: "#665533"})

	zoneLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#333333", Dark: "#F8F8F2"})
)

// UpdateColors updates all styles with config colors
func UpdateColors(primary, secondary, accent, errorColor, success string) {
	colorPrimary = lipgloss.Color(primary)
	colorSecondary = lipgloss.Color(secondary)
	colorAccent = lipgloss.Color(accent)
	colorError = lipgloss.Color(errorColor)
	colorSuccess = lipgloss.Color(success)

	// Update all styles (reapply colors)
	titleStyle = titleStyle.Foreground(colorPrimary)
	headerStyle = headerStyle.Foreground(colorSecondary)
	selectedStyle = selectedStyle.Foreground(colorAccent)
	favoriteStyle = favoriteStyle.Foreground(colorAccent)
	liveStyle = liveStyle.Foreground(colorError)
	boxStyle = boxStyle.BorderForeground(colorPrimary)
	inningStyle = inningStyle.Foreground(colorSecondary)
	countFilledStyle = countFilledStyle.Foreground(colorAccent)
	strikeFilledStyle = strikeFilledStyle.Foreground(colorError)
	outFilledStyle = outFilledStyle.Foreground(colorError)
	baseFilledStyle = baseFilledStyle.Foreground(colorAccent)
	tableHeaderStyle = tableHeaderStyle.Foreground(colorSecondary)
	zoneBorderStyle = zoneBorderStyle.Foreground(colorSubtle)
	zoneBallStyle = zoneBallStyle.Foreground(colorAccent)
	zoneStrikeStyle = zoneStrikeStyle.Foreground(colorError)
	zoneInPlayStyle = zoneInPlayStyle.Foreground(colorSecondary)
}

// UpdateEventColors updates event color variables from config
func UpdateEventColors(cfg config.EventColorConfig) {
	colorInningHeader = lipgloss.Color(cfg.InningHeader)
	colorStrikeout = lipgloss.Color(cfg.Strikeout)
	colorWalk = lipgloss.Color(cfg.Walk)
	colorInPlayNoOut = lipgloss.Color(cfg.InPlayNoOut)
	colorInPlayOut = lipgloss.Color(cfg.InPlayOut)
	colorDefaultEvent = lipgloss.Color(cfg.DefaultEvent)
	colorActionEvent = lipgloss.Color(cfg.ActionEvent)
	colorScoringPlay = lipgloss.Color(cfg.ScoringPlay)
	colorScoreBadgeFg = lipgloss.Color(cfg.ScoreBadgeFg)
	colorScoreBadgeBg = lipgloss.Color(cfg.ScoreBadgeBg)
	colorLiveInning = lipgloss.Color(cfg.LiveInning)
}

// RenderGradientTitle renders a title with a gradient from primary to accent color
func RenderGradientTitle(text string) string {
	// Create gradient using lipgloss
	return gradientTitleStyle.
		Foreground(colorPrimary).
		Render(text)
}
