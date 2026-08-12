package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pdavlin/go-playball/internal/api"
	"github.com/pdavlin/go-playball/internal/ui/anim"
)

// hotBatsRowsPerTeam is the number of hitter rows per team table.
const hotBatsRowsPerTeam = 3

// renderHotBatsCard renders the Hot Bats tab body: a window selector
// header on top, then two team tables.
func renderHotBatsCard(
	game *api.Game,
	data *pregameGameData,
	window HotBatsWindow,
	spinner *anim.Spinner,
	width int,
) string {
	if game == nil || game.GameData == nil {
		return itemStyle.Render("Game data unavailable")
	}

	awayName := game.GameData.Teams.Away.Name
	homeName := game.GameData.Teams.Home.Name
	awayColors := GetTeamColors(awayName)
	homeColors := GetTeamColors(homeName)

	header := renderHotBatsHeader(window, homeColors.Primary)

	awayPayload, homePayload := pickHotBatsTeams(data, window)

	awayCol := renderHotBatsTeam(GetTeamShortName(awayName), awayColors, awayPayload, spinner)
	homeCol := renderHotBatsTeam(GetTeamShortName(homeName), homeColors, homePayload, spinner)

	body := renderTwoTeamColumns(awayCol, homeCol, width)
	return lipgloss.JoinVertical(lipgloss.Left, header, "", body)
}

// pickHotBatsTeams returns the per-team payload pointers for the active
// window, or nil pointers when the cache hasn't been populated yet.
func pickHotBatsTeams(data *pregameGameData, window HotBatsWindow) (*hotBatsTeamPayload, *hotBatsTeamPayload) {
	if data == nil || data.hotBats == nil {
		return nil, nil
	}
	return data.hotBats.awayByWindow[window], data.hotBats.homeByWindow[window]
}

// renderHotBatsHeader renders the window chip strip: `[ L7 ]  L15   L30`.
// The active window is bracketed and bold.
func renderHotBatsHeader(active HotBatsWindow, selectedColor lipgloss.TerminalColor) string {
	// The window selector speaks the same underline language as the tab
	// strips, but its rule stops at the chip row's edge so it reads as a
	// local control, not a section divider.
	entries := make([]stripEntry, 0, 3)
	for _, w := range []HotBatsWindow{HotBatsL7, HotBatsL15, HotBatsL30} {
		entries = append(entries, stripEntry{label: w.label(), selected: w == active})
	}
	return renderUnderlineChipRow(entries, selectedColor)
}

// renderHotBatsTeam renders one team's section: team name header + the
// hitter table (or skeleton/fallbacks).
func renderHotBatsTeam(
	teamName string,
	colors TeamColors,
	payload *hotBatsTeamPayload,
	spinner *anim.Spinner,
) string {
	var b strings.Builder
	b.WriteString(renderPregameTeamHeader(teamName, colors))

	switch {
	case payload == nil || !payload.loaded:
		b.WriteString(hotBatsTableHeader())
		b.WriteString("\n")
		b.WriteString(renderHotBatsSkeletonRows(spinner))
	case payload.err != nil:
		b.WriteString(itemStyle.Render("Data unavailable"))
	default:
		b.WriteString(hotBatsTableHeader())
		b.WriteString("\n")
		b.WriteString(renderHotBatsRows(payload.rows))
	}

	return b.String()
}

func hotBatsTableHeader() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}).
		Bold(true).
		Render(fmt.Sprintf("%-22s %5s %3s %4s %6s",
			"Name", "AVG", "HR", "RBI", "OPS"))
}

// renderHotBatsRows renders up to hotBatsRowsPerTeam rows. Empty slots
// (insufficient data) render "Insufficient data" once, not per row.
func renderHotBatsRows(rows []hotBatsRow) string {
	if len(rows) == 0 {
		return itemStyle.Render("Insufficient data")
	}
	var b strings.Builder
	for i, r := range rows {
		if i >= hotBatsRowsPerTeam {
			break
		}
		nameCell := truncatePitchName(r.name, 22)
		line := fmt.Sprintf("%-22s %5s %3d %4d %6s",
			nameCell, r.avg, r.homeRuns, r.rbi, r.ops)
		b.WriteString(line)
		b.WriteString("\n")
	}
	// Pad missing rows with a single dim notice rather than spinner
	// cells — at this point the fetch resolved.
	if len(rows) < hotBatsRowsPerTeam {
		b.WriteString(itemStyle.Render("(insufficient data for remaining slots)"))
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderHotBatsSkeletonRows(spinner *anim.Spinner) string {
	var b strings.Builder
	for i := 0; i < hotBatsRowsPerTeam; i++ {
		row := skeletonCell(spinner, 22) + " " +
			skeletonCell(spinner, 5) + " " +
			skeletonCell(spinner, 3) + " " +
			skeletonCell(spinner, 4) + " " +
			skeletonCell(spinner, 6)
		b.WriteString(row)
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
