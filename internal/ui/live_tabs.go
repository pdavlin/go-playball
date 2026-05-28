package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pdavlin/go-playball/internal/api"
)

type tabSpec struct {
	key   string
	label string
	tab   LiveTab
}

// availableLiveTabs returns the tabs visible in the right column.
// All tabs share the column with Plays, so if Plays fits, they fit.
func (m Model) availableLiveTabs() []tabSpec {
	return []tabSpec{
		{"1", "Plays", LiveTabPlays},
		{"2", "Mix", LiveTabPitchMix},
	}
}

// pitchingTeamPrimary picks the team color tied to whichever team is
// currently pitching. Falls back to the home color when no inning
// state is available.
func pitchingTeamPrimary(game *api.Game) lipgloss.Color {
	away, home := getGameTeamColorsFull(game)
	if game.LiveData != nil {
		switch game.LiveData.Linescore.InningState {
		case "Bottom", "End":
			return away.Primary
		}
	}
	return home.Primary
}

// renderTabStrip renders the one-line tab strip. The selected tab is
// bold and tinted with the pitching team's primary color; unselected
// tabs are dim.
func (m Model) renderTabStrip(width int, selectedColor lipgloss.TerminalColor) string {
	tabs := m.availableLiveTabs()

	selectedStyle := lipgloss.NewStyle().
		Foreground(selectedColor).
		Bold(true)
	unselectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#888888", Dark: "#666666"})

	var parts []string
	for _, t := range tabs {
		label := "[" + t.key + "] " + t.label
		if t.tab == m.liveTab {
			parts = append(parts, selectedStyle.Render(label))
		} else {
			parts = append(parts, unselectedStyle.Render(label))
		}
	}

	strip := strings.Join(parts, "  ")
	return lipgloss.NewStyle().Width(width).Render(strip)
}

// renderLiveRightColumn picks which card to render based on the
// active LiveTab and prepends a one-line tab strip.
func (m Model) renderLiveRightColumn(game *api.Game, height, width int) string {
	strip := m.renderTabStrip(width, pitchingTeamPrimary(game))

	bodyHeight := height - 1
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	var body string
	switch m.liveTab {
	case LiveTabPitchMix:
		body = renderPitchMixCard(game, bodyHeight, width)
	default:
		body = m.renderPlays(game, bodyHeight, width, false, true)
	}

	return lipgloss.JoinVertical(lipgloss.Left, strip, body)
}
