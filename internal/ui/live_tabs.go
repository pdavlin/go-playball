package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pdavlin/go-playball/internal/api"
)

type tabSpec struct {
	label string
	tab   LiveTab
}

// availableLiveTabs returns the tabs visible in the right column.
// All tabs share the column with Plays, so if Plays fits, they fit.
// Number keys 1-3 select tabs directly; the help bar documents them.
func (m Model) availableLiveTabs() []tabSpec {
	return []tabSpec{
		{"Plays", LiveTabPlays},
		{"Pitch Mix", LiveTabPitchMix},
		{"Win Prob", LiveTabWinProb},
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

// stripEntry is one label cell in a tab strip. Shared by the live and
// pregame strips so both render identically.
type stripEntry struct {
	label    string
	selected bool
}

// Tab strip layout constants. The indent matches pregameLeftGutter so
// strip labels line up with guttered card bodies; the underline rule
// doubles as the strip-to-body separator in both views.
const (
	tabStripIndent = 2
	tabStripGap    = "   "
)

// tabStripDimStyle is the shared dim tone for unselected labels and the
// hairline segments of the underline rule.
var tabStripDimStyle = lipgloss.NewStyle().
	Foreground(lipgloss.AdaptiveColor{Light: "#888888", Dark: "#666666"})

// renderUnderlineTabStrip renders a two-line tab strip: a row of flat
// labels, then a full-width hairline rule with a heavy, tinted segment
// under the selected label. The rule separates the strip from the body,
// so callers should not add their own separator.
func renderUnderlineTabStrip(entries []stripEntry, width int, selectedColor lipgloss.TerminalColor) string {
	selectedStyle := lipgloss.NewStyle().
		Foreground(selectedColor).
		Bold(true)

	var labels strings.Builder
	var rule strings.Builder
	labels.WriteString(strings.Repeat(" ", tabStripIndent))
	rule.WriteString(tabStripDimStyle.Render(strings.Repeat("─", tabStripIndent)))
	ruleWidth := tabStripIndent

	for i, e := range entries {
		if i > 0 {
			labels.WriteString(tabStripGap)
			rule.WriteString(tabStripDimStyle.Render(strings.Repeat("─", len(tabStripGap))))
			ruleWidth += len(tabStripGap)
		}
		w := lipgloss.Width(e.label)
		if e.selected {
			labels.WriteString(selectedStyle.Render(e.label))
			rule.WriteString(selectedStyle.Render(strings.Repeat("━", w)))
		} else {
			labels.WriteString(tabStripDimStyle.Render(e.label))
			rule.WriteString(tabStripDimStyle.Render(strings.Repeat("─", w)))
		}
		ruleWidth += w
	}
	if remaining := width - ruleWidth; remaining > 0 {
		rule.WriteString(tabStripDimStyle.Render(strings.Repeat("─", remaining)))
	}

	labelLine := lipgloss.NewStyle().Width(width).Render(labels.String())
	return labelLine + "\n" + rule.String()
}

// renderUnderlineChipRow renders a compact selector in the same
// underline language as the tab strips, but the rule stops at the last
// chip instead of running the full width. Used for tab-local options
// like the Hot Bats window selector.
func renderUnderlineChipRow(entries []stripEntry, selectedColor lipgloss.TerminalColor) string {
	selectedStyle := lipgloss.NewStyle().
		Foreground(selectedColor).
		Bold(true)

	var labels strings.Builder
	var rule strings.Builder
	labels.WriteString(strings.Repeat(" ", tabStripIndent))
	rule.WriteString(strings.Repeat(" ", tabStripIndent))

	for i, e := range entries {
		if i > 0 {
			labels.WriteString(tabStripGap)
			rule.WriteString(strings.Repeat(" ", len(tabStripGap)))
		}
		w := lipgloss.Width(e.label)
		if e.selected {
			labels.WriteString(selectedStyle.Render(e.label))
			rule.WriteString(selectedStyle.Render(strings.Repeat("━", w)))
		} else {
			labels.WriteString(tabStripDimStyle.Render(e.label))
			rule.WriteString(strings.Repeat(" ", w))
		}
	}
	return labels.String() + "\n" + rule.String()
}

// renderTabStrip renders the live-game tab strip.
func (m Model) renderTabStrip(width int, selectedColor lipgloss.TerminalColor) string {
	tabs := m.availableLiveTabs()
	entries := make([]stripEntry, 0, len(tabs))
	for _, t := range tabs {
		entries = append(entries, stripEntry{
			label:    t.label,
			selected: t.tab == m.liveTab,
		})
	}
	return renderUnderlineTabStrip(entries, width, selectedColor)
}

// renderLiveRightColumn picks which card to render based on the
// active LiveTab and prepends a one-line tab strip. Plays manages its
// own scrolling viewport; the other cards return natural-height
// content and share the tab-body viewport with the pregame tabs.
func (m Model) renderLiveRightColumn(game *api.Game, height, width int) string {
	strip := m.renderTabStrip(width, pitchingTeamPrimary(game))

	// The strip is two lines: labels + underline rule.
	bodyHeight := height - 2
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	var body string
	switch m.liveTab {
	case LiveTabPitchMix:
		body = renderPitchMixCard(game, width)
		body = renderTabBodyViewport(game, body, width, bodyHeight, m.gameScrollOffset)
	case LiveTabWinProb:
		body = renderWinProbCard(game, m.winProb, bodyHeight, width)
		body = renderTabBodyViewport(game, body, width, bodyHeight, m.gameScrollOffset)
	default:
		body = m.renderPlays(game, bodyHeight, width, false, true)
	}

	return lipgloss.JoinVertical(lipgloss.Left, strip, body)
}
