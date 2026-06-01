package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pdavlin/go-playball/internal/api"
)

type pregameTabSpec struct {
	key     string
	label   string
	tab     PregameTab
	enabled bool
}

// availablePregameTabs returns the tab strip for a Preview-state game.
// The 3-column overview is always rendered above the strip, so there is
// no dedicated Overview tab. Disabled entries render dim and their key
// bindings no-op.
func availablePregameTabs() []pregameTabSpec {
	return []pregameTabSpec{
		{"1", "Pitchers", PregameTabPitchers, true},
		{"2", "Hot Bats", PregameTabHotBats, true},
		{"3", "H2H", PregameTabH2H, true},
		{"4", "Bullpen", PregameTabBullpen, true},
	}
}

// renderPregameTabStrip renders the one-line tab strip. Selected tab is
// bold and tinted with selectedColor; unselected enabled tabs are dim;
// disabled tabs are dimmer with a trailing "·" suffix.
func (m Model) renderPregameTabStrip(width int, selectedColor lipgloss.TerminalColor) string {
	selectedStyle := lipgloss.NewStyle().
		Foreground(selectedColor).
		Bold(true)
	unselectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#888888", Dark: "#666666"})
	disabledStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#BBBBBB", Dark: "#444444"})

	var parts []string
	for _, t := range availablePregameTabs() {
		label := "[" + t.key + "] " + t.label
		switch {
		case t.tab == m.pregameTab && t.enabled:
			parts = append(parts, selectedStyle.Render(label))
		case !t.enabled:
			parts = append(parts, disabledStyle.Render(label+"·"))
		default:
			parts = append(parts, unselectedStyle.Render(label))
		}
	}

	strip := strings.Join(parts, "  ")
	return lipgloss.NewStyle().Width(width).Render(strip)
}

// renderPregameTabBody dispatches to the body renderer for the active tab.
// Overview returns an empty string because the 3-column overview is
// always rendered above the tab strip.
func (m Model) renderPregameTabBody(game *api.Game, width, height int) string {
	switch m.pregameTab {
	case PregameTabPitchers:
		return renderPitcherDetailCard(game, m.pregameData[game.ID], m.pregameSpinner, width, height)
	case PregameTabHotBats:
		return renderHotBatsCard(game, m.pregameData[game.ID], m.hotBatsWindow, m.pregameSpinner, width, height)
	case PregameTabH2H:
		return renderH2HCard(game, m.pregameData[game.ID], m.pregameSpinner, width, height)
	case PregameTabBullpen:
		return renderBullpenCard(game, m.pregameData[game.ID], m.pregameSpinner, width, height)
	default:
		return ""
	}
}
