package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pdavlin/go-playball/internal/api"
	"github.com/pdavlin/go-playball/internal/ui/anim"
)

const bullpenSideBySideWidth = 110

// renderBullpenCard renders the Bullpen Workload tab body.
func renderBullpenCard(
	game *api.Game,
	data *pregameGameData,
	spinner *anim.Spinner,
	width, height int,
) string {
	if game == nil || game.GameData == nil {
		return padToHeight(itemStyle.Render("Game data unavailable"), height)
	}

	const leftGutter = 2
	usable := width - leftGutter

	awayName := game.GameData.Teams.Away.Name
	homeName := game.GameData.Teams.Home.Name
	awayColors := GetTeamColors(awayName)
	homeColors := GetTeamColors(homeName)

	var awayPayload, homePayload *bullpenTeamPayload
	if data != nil && data.bullpen != nil {
		awayPayload = data.bullpen.away
		homePayload = data.bullpen.home
	}

	awayCol := renderBullpenTeam(GetTeamShortName(awayName), awayColors, awayPayload, spinner)
	homeCol := renderBullpenTeam(GetTeamShortName(homeName), homeColors, homePayload, spinner)

	var body string
	if usable >= bullpenSideBySideWidth {
		colWidth := (usable - 4) / 2
		if colWidth < 40 {
			colWidth = 40
		}
		left := lipgloss.NewStyle().Width(colWidth).Render(awayCol)
		gap := "  "
		right := lipgloss.NewStyle().Width(colWidth).Render(homeCol)
		body = lipgloss.JoinHorizontal(lipgloss.Top, left, gap, right)
	} else {
		body = lipgloss.JoinVertical(lipgloss.Left, awayCol, "", homeCol)
	}

	footer := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}).
		Render(fmt.Sprintf(
			"Ready: %d+ days rest  |  Limited: pitched yesterday <%dP  |  Unavail: yesterday %d+P",
			bullpenRecentDaysWindow+1, bullpenUnavailPitchThreshold, bullpenUnavailPitchThreshold,
		))

	full := lipgloss.JoinVertical(lipgloss.Left, body, "", footer)
	full = lipgloss.NewStyle().PaddingLeft(leftGutter).Render(full)
	return padToHeight(full, height)
}

// renderBullpenTeam renders one team's section: header + reliever list.
func renderBullpenTeam(
	teamName string,
	colors TeamColors,
	payload *bullpenTeamPayload,
	spinner *anim.Spinner,
) string {
	var b strings.Builder
	nameStyle := lipgloss.NewStyle().Foreground(colors.Primary).Bold(true)
	b.WriteString(nameStyle.Render(teamName))
	b.WriteString("\n\n")

	switch {
	case payload == nil || !payload.loaded:
		b.WriteString(renderBullpenSkeleton(spinner))
	case payload.err != nil:
		b.WriteString(itemStyle.Render("Bullpen data unavailable"))
	case len(payload.relievers) == 0:
		b.WriteString(itemStyle.Render("No relievers on roster"))
	default:
		b.WriteString(renderBullpenRelievers(payload.relievers))
	}
	return b.String()
}

// renderBullpenRelievers renders the per-reliever lines + sub-rows.
func renderBullpenRelievers(rows []bullpenReliever) string {
	var b strings.Builder
	subStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#888888", Dark: "#888888"})
	for i, r := range rows {
		if i > 0 {
			b.WriteString("\n")
		}
		dot := renderBullpenStatusDot(r.status)
		summary := bullpenSummary(r)
		b.WriteString(fmt.Sprintf("%s %-20s %s", dot, truncatePitchName(r.name, 20), summary))
		b.WriteString("\n")
		if len(r.appearances) == 0 {
			b.WriteString(subStyle.Render("   No recent appearances"))
			b.WriteString("\n")
			continue
		}
		for _, a := range r.appearances {
			line := fmt.Sprintf("   %-10s  %s IP, %dP", a.date, a.inningsPitched, a.pitches)
			b.WriteString(subStyle.Render(line))
			b.WriteString("\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// bullpenSummary returns the right-of-name summary string, e.g.
// `2 of 3 - 58P` (appearances in last block, total pitches).
func bullpenSummary(r bullpenReliever) string {
	if len(r.appearances) == 0 {
		return ""
	}
	total := 0
	for _, a := range r.appearances {
		total += a.pitches
	}
	return fmt.Sprintf("%d of %d - %dP",
		len(r.appearances), bullpenRecentAppearances, total)
}

// renderBullpenStatusDot returns a colored dot for the status. Green
// for Ready, yellow for Limited, red for Unavail.
func renderBullpenStatusDot(s bullpenStatusKind) string {
	var color lipgloss.TerminalColor
	switch s {
	case bullpenUnavail:
		color = lipgloss.AdaptiveColor{Light: "#CC0000", Dark: "#FF5555"}
	case bullpenLimited:
		color = lipgloss.AdaptiveColor{Light: "#B58900", Dark: "#F1FA8C"}
	default:
		color = lipgloss.AdaptiveColor{Light: "#008000", Dark: "#50FA7B"}
	}
	return lipgloss.NewStyle().Foreground(color).Render("●")
}

// renderBullpenSkeleton renders spinner-filled placeholder rows.
func renderBullpenSkeleton(spinner *anim.Spinner) string {
	var b strings.Builder
	subStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#888888", Dark: "#888888"})
	dimDot := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#BBBBBB", Dark: "#444444"}).
		Render("●")

	for i := 0; i < 4; i++ {
		b.WriteString(fmt.Sprintf("%s %s %s\n",
			dimDot,
			skeletonCell(spinner, 20),
			skeletonCell(spinner, 12),
		))
		for j := 0; j < 2; j++ {
			b.WriteString(subStyle.Render("   "))
			b.WriteString(skeletonCell(spinner, 10))
			b.WriteString("  ")
			b.WriteString(skeletonCell(spinner, 12))
			b.WriteString("\n")
		}
		if i < 3 {
			b.WriteString("\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}
