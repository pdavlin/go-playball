package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pdavlin/go-playball/internal/api"
	"github.com/pdavlin/go-playball/internal/ui/anim"
)

// hotBatsSideBySideWidth mirrors the pitcher card: two columns above
// this width, stacked below.
const hotBatsSideBySideWidth = 110

// hotBatsRowsPerTeam is the number of hitter rows per team table.
const hotBatsRowsPerTeam = 3

// renderHotBatsCard renders the Hot Bats tab body: a window selector
// header on top, then two team tables.
func renderHotBatsCard(
	game *api.Game,
	data *pregameGameData,
	window HotBatsWindow,
	spinner *anim.Spinner,
	width, height int,
) string {
	if game == nil || game.GameData == nil {
		return padToHeight(itemStyle.Render("Game data unavailable"), height)
	}

	header := renderHotBatsHeader(window, width)

	awayName := game.GameData.Teams.Away.Name
	homeName := game.GameData.Teams.Home.Name
	awayColors := GetTeamColors(awayName)
	homeColors := GetTeamColors(homeName)

	awayPayload, homePayload := pickHotBatsTeams(data, window)

	awayCol := renderHotBatsTeam(GetTeamShortName(awayName), awayColors, awayPayload, spinner)
	homeCol := renderHotBatsTeam(GetTeamShortName(homeName), homeColors, homePayload, spinner)

	const leftGutter = 2
	usable := width - leftGutter

	var body string
	if usable >= hotBatsSideBySideWidth {
		colWidth := (usable - 4) / 2
		if colWidth < 36 {
			colWidth = 36
		}
		left := lipgloss.NewStyle().Width(colWidth).Render(awayCol)
		gap := "  "
		right := lipgloss.NewStyle().Width(colWidth).Render(homeCol)
		body = lipgloss.JoinHorizontal(lipgloss.Top, left, gap, right)
	} else {
		body = lipgloss.JoinVertical(lipgloss.Left, awayCol, "", homeCol)
	}
	body = lipgloss.NewStyle().PaddingLeft(leftGutter).Render(body)

	full := lipgloss.JoinVertical(lipgloss.Left, header, "", body)
	return padToHeight(full, height)
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
func renderHotBatsHeader(active HotBatsWindow, width int) string {
	const leftGutter = 2
	activeStyle := lipgloss.NewStyle().Bold(true)
	inactiveStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#888888", Dark: "#666666"})

	var parts []string
	for _, w := range []HotBatsWindow{HotBatsL7, HotBatsL15, HotBatsL30} {
		label := w.label()
		if w == active {
			parts = append(parts, activeStyle.Render("[ "+label+" ]"))
		} else {
			parts = append(parts, inactiveStyle.Render("  "+label+"  "))
		}
	}
	strip := strings.Join(parts, " ")
	return lipgloss.NewStyle().PaddingLeft(leftGutter).Width(width).Render(strip)
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
	nameStyle := lipgloss.NewStyle().Foreground(colors.Primary).Bold(true)
	b.WriteString(nameStyle.Render(teamName))
	b.WriteString("\n\n")

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
