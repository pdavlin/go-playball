package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pdavlin/go-playball/internal/api"
	"github.com/pdavlin/go-playball/internal/ui/anim"
)

type pregameTabSpec struct {
	key   string
	label string
	tab   PregameTab
}

// Shared frame constants for all pregame tab cards. Every card uses the
// same left gutter and flips between side-by-side and stacked team
// columns at the same width, so switching tabs never shifts the layout.
const (
	pregameLeftGutter      = 2
	pregameSideBySideWidth = 110
	pregameTwoColumnGap    = "  "
)

// availablePregameTabs returns the tab strip for a Preview-state game.
// The 3-column overview is always rendered above the strip, so there is
// no dedicated Overview tab.
func availablePregameTabs() []pregameTabSpec {
	return []pregameTabSpec{
		{"1", "Pitchers", PregameTabPitchers},
		{"2", "Hot Bats", PregameTabHotBats},
		{"3", "H2H", PregameTabH2H},
		{"4", "Bullpen", PregameTabBullpen},
	}
}

// renderPregameTabStrip renders the one-line tab strip in the same
// visual language as the live-game strip.
func (m Model) renderPregameTabStrip(width int, selectedColor lipgloss.TerminalColor) string {
	entries := make([]stripEntry, 0, 4)
	for _, t := range availablePregameTabs() {
		entries = append(entries, stripEntry{
			key:      t.key,
			label:    t.label,
			selected: t.tab == m.pregameTab,
		})
	}
	return renderKeyedTabStrip(entries, width, selectedColor)
}

// renderPregameTabBody dispatches to the body renderer for the active
// tab. Cards return natural-height content; the caller fits it to the
// height budget via renderPregameViewport.
func (m Model) renderPregameTabBody(game *api.Game, width int) string {
	switch m.pregameTab {
	case PregameTabHotBats:
		return renderHotBatsCard(game, m.pregameData[game.ID], m.hotBatsWindow, m.pregameSpinner, width)
	case PregameTabH2H:
		return renderH2HCard(game, m.pregameData[game.ID], m.pregameSpinner, width)
	case PregameTabBullpen:
		return renderBullpenCard(game, m.pregameData[game.ID], m.pregameSpinner, width)
	default:
		return renderPitcherDetailCard(game, m.pregameData[game.ID], m.pregameSpinner, width)
	}
}

// renderPregameViewport fits a tab body into the height budget. Short
// content is padded so the help bar stays pinned; overflow scrolls with
// the same team-colored indicators as the plays viewport. offset is
// clamped to the content bounds.
func renderPregameViewport(game *api.Game, body string, width, height, offset int) string {
	if height <= 0 {
		return ""
	}
	lines := strings.Split(body, "\n")
	if len(lines) <= height && offset <= 0 {
		return padToHeight(body, height)
	}

	reserved := 1 // bottom indicator slot whenever content can scroll
	if offset > 0 {
		reserved++
	}
	viewHeight := height - reserved
	if viewHeight < 1 {
		viewHeight = 1
	}
	maxOffset := len(lines) - viewHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	if offset < 0 {
		offset = 0
	}
	viewEnd := offset + viewHeight
	if viewEnd > len(lines) {
		viewEnd = len(lines)
	}

	away, home := getGameTeamColors(game)
	var b strings.Builder
	if offset > 0 {
		b.WriteString(anim.ScrollIndicator(anim.ScrollUp, offset, width, away, home))
		b.WriteString("\n")
	}
	b.WriteString(strings.Join(lines[offset:viewEnd], "\n"))
	if viewEnd < len(lines) {
		b.WriteString("\n")
		b.WriteString(anim.ScrollIndicator(anim.ScrollDown, len(lines)-viewEnd, width, away, home))
	}
	return padToHeight(b.String(), height)
}

// renderTwoTeamColumns joins the away/home sections side-by-side when
// the card is wide enough, stacked otherwise, and applies the shared
// left gutter. Used by every two-team pregame card.
func renderTwoTeamColumns(awayCol, homeCol string, width int) string {
	usable := width - pregameLeftGutter
	var body string
	if usable >= pregameSideBySideWidth {
		colWidth := (usable - len(pregameTwoColumnGap) - 2) / 2
		left := lipgloss.NewStyle().Width(colWidth).Render(awayCol)
		right := lipgloss.NewStyle().Width(colWidth).Render(homeCol)
		body = lipgloss.JoinHorizontal(lipgloss.Top, left, pregameTwoColumnGap, right)
	} else {
		body = lipgloss.JoinVertical(lipgloss.Left, awayCol, "", homeCol)
	}
	return lipgloss.NewStyle().PaddingLeft(pregameLeftGutter).Render(body)
}

// renderPregameTeamHeader renders the team-name section header used by
// the two-team cards: short name, primary color, bold, blank line after.
func renderPregameTeamHeader(teamName string, colors TeamColors) string {
	return lipgloss.NewStyle().Foreground(colors.Primary).Bold(true).
		Render(teamName) + "\n\n"
}
