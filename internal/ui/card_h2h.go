package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pdavlin/go-playball/internal/api"
	"github.com/pdavlin/go-playball/internal/ui/anim"
)

// renderH2HCard renders the Head-to-Head tab body.
func renderH2HCard(
	game *api.Game,
	data *pregameGameData,
	spinner *anim.Spinner,
	width int,
) string {
	if game == nil || game.GameData == nil {
		return itemStyle.Render("Game data unavailable")
	}

	usable := width - pregameLeftGutter

	awayName := game.GameData.Teams.Away.Name
	homeName := game.GameData.Teams.Home.Name
	awayAbbr := getTeamAbbreviation(awayName)
	homeAbbr := getTeamAbbreviation(homeName)
	awayColors := GetTeamColors(awayName)
	homeColors := GetTeamColors(homeName)

	var body string
	switch {
	case data == nil || data.h2h == nil || !data.h2h.loaded:
		body = renderH2HSkeleton(awayAbbr, homeAbbr, awayColors, homeColors, spinner, usable)
	case data.h2h.err != nil:
		body = itemStyle.Render("Series data unavailable")
	case data.h2h.games == 0:
		body = itemStyle.Render("No prior meetings this season")
	default:
		body = renderH2HBody(data.h2h, awayAbbr, homeAbbr, awayColors, homeColors, usable)
	}

	return lipgloss.NewStyle().PaddingLeft(pregameLeftGutter).Render(body)
}

// renderH2HBody renders the full series block once data is loaded.
func renderH2HBody(
	p *h2hPayload,
	awayAbbr, homeAbbr string,
	awayColors, homeColors TeamColors,
	width int,
) string {
	var b strings.Builder

	b.WriteString(renderH2HSeriesHeader(p, awayAbbr, homeAbbr, awayColors, homeColors))
	b.WriteString("\n")
	b.WriteString(renderH2HBipolarBar(p, awayColors, homeColors, width))
	b.WriteString("\n\n")
	b.WriteString(renderH2HLastMeeting(p, awayAbbr, homeAbbr))
	b.WriteString("\n\n")
	b.WriteString(renderH2HStatGrid(p, awayAbbr, homeAbbr))

	return b.String()
}

// renderH2HSeriesHeader renders `{AWAY} {wins}  Series ({games}G)  {wins} {HOME}`.
func renderH2HSeriesHeader(
	p *h2hPayload,
	awayAbbr, homeAbbr string,
	awayColors, homeColors TeamColors,
) string {
	awayLabel := lipgloss.NewStyle().Foreground(awayColors.Primary).Bold(true).
		Render(fmt.Sprintf("%s %d", awayAbbr, p.awayWins))
	homeLabel := lipgloss.NewStyle().Foreground(homeColors.Primary).Bold(true).
		Render(fmt.Sprintf("%d %s", p.homeWins, homeAbbr))
	mid := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}).
		Render(fmt.Sprintf("  Series (%dG)  ", p.games))
	return awayLabel + mid + homeLabel
}

// renderH2HBipolarBar renders a proportional bar split between the two
// teams. Total wins == games (ignoring ties, which MLB has none of).
// A thin contrasting marker always separates the two fills so the bar
// reads as a ratio rather than a solid divider; a shut-out side (0
// wins) still gets a dim sliver so it visibly reads as "all one team"
// instead of disappearing entirely.
func renderH2HBipolarBar(p *h2hPayload, awayColors, homeColors TeamColors, width int) string {
	barWidth := width - 4
	if barWidth < 10 {
		barWidth = 10
	}
	total := p.awayWins + p.homeWins
	if total == 0 {
		return strings.Repeat("─", barWidth)
	}

	// Reserve one cell for the split marker between the two fills.
	fillWidth := barWidth - 1
	if fillWidth < 2 {
		fillWidth = 2
	}

	awayCells := int(float64(p.awayWins) / float64(total) * float64(fillWidth))
	if awayCells < 0 {
		awayCells = 0
	}
	if awayCells > fillWidth {
		awayCells = fillWidth
	}
	homeCells := fillWidth - awayCells

	dimColor := lipgloss.AdaptiveColor{Light: "#CCCCCC", Dark: "#444444"}
	dimAway, dimHome := false, false
	if p.awayWins == 0 && awayCells == 0 && homeCells > 1 {
		awayCells = 1
		homeCells--
		dimAway = true
	} else if p.homeWins == 0 && homeCells == 0 && awayCells > 1 {
		homeCells = 1
		awayCells--
		dimHome = true
	}

	awayStyle := lipgloss.NewStyle().Foreground(awayColors.Primary)
	if dimAway {
		awayStyle = lipgloss.NewStyle().Foreground(dimColor)
	}
	homeStyle := lipgloss.NewStyle().Foreground(homeColors.Primary)
	if dimHome {
		homeStyle = lipgloss.NewStyle().Foreground(dimColor)
	}
	markerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#333333", Dark: "#EEEEEE"})

	awayFill := awayStyle.Render(strings.Repeat("█", awayCells))
	homeFill := homeStyle.Render(strings.Repeat("█", homeCells))
	marker := markerStyle.Render("│")

	return awayFill + marker + homeFill
}

func renderH2HLastMeeting(p *h2hPayload, awayAbbr, homeAbbr string) string {
	label := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}).
		Render("Last meeting: ")
	if p.lastMeeting == nil {
		return label + "—"
	}
	value := fmt.Sprintf("%s  %s %d - %d %s",
		p.lastMeeting.date,
		awayAbbr, p.lastMeeting.awayRuns,
		p.lastMeeting.homeRuns, homeAbbr,
	)
	return label + value
}

// renderH2HStatGrid lays out the six aggregate stats in a 2-row, 3-col
// grid. Each cell is "label\nvalue". The grid is dimmed labels + bold
// values.
func renderH2HStatGrid(p *h2hPayload, awayAbbr, homeAbbr string) string {
	games := p.games
	avgAway := "0.0"
	avgHome := "0.0"
	avgTotal := "0.0"
	if games > 0 {
		avgAway = fmt.Sprintf("%.1f", float64(p.awayRunsTotal)/float64(games))
		avgHome = fmt.Sprintf("%.1f", float64(p.homeRunsTotal)/float64(games))
		avgTotal = fmt.Sprintf("%.1f", float64(p.awayRunsTotal+p.homeRunsTotal)/float64(games))
	}

	runDiff := p.awayRunsTotal - p.homeRunsTotal
	diffStr := "Even"
	if runDiff != 0 {
		// Render from the away team's perspective: +N means away is up.
		sign := "+"
		val := runDiff
		if val < 0 {
			sign = "-"
			val = -val
		}
		diffStr = fmt.Sprintf("%s%d %s", sign, val, awayAbbr)
		if runDiff < 0 {
			diffStr = fmt.Sprintf("+%d %s", val, homeAbbr)
		}
	}

	cells := []struct {
		label, value string
	}{
		{"Runs", fmt.Sprintf("%d-%d", p.awayRunsTotal, p.homeRunsTotal)},
		{"Run diff", diffStr},
		{"Avg score", fmt.Sprintf("%s-%s", avgAway, avgHome)},
		{"Avg total", fmt.Sprintf("%s R/G", avgTotal)},
		{"1-run games", fmt.Sprintf("%d", p.oneRunGames)},
		{"Largest margin", fmt.Sprintf("%d", p.largestMargin)},
	}

	cellWidth := 22
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"})
	valueStyle := lipgloss.NewStyle().Bold(true)

	var row1, row2 strings.Builder
	for i, c := range cells {
		cell := labelStyle.Render(c.label) + "\n" + valueStyle.Render(c.value)
		col := lipgloss.NewStyle().Width(cellWidth).Render(cell)
		if i < 3 {
			row1.WriteString(col)
		} else {
			row2.WriteString(col)
		}
	}

	return row1.String() + "\n" + row2.String()
}

// renderH2HSkeleton renders the loading state in the table shape so
// the layout doesn't pop in when data arrives.
func renderH2HSkeleton(
	awayAbbr, homeAbbr string,
	awayColors, homeColors TeamColors,
	spinner *anim.Spinner,
	width int,
) string {
	awayLabel := lipgloss.NewStyle().Foreground(awayColors.Primary).Bold(true).
		Render(awayAbbr + " ")
	homeLabel := lipgloss.NewStyle().Foreground(homeColors.Primary).Bold(true).
		Render(" " + homeAbbr)
	mid := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}).
		Render("  Series  ")

	header := awayLabel + skeletonCell(spinner, 2) + mid + skeletonCell(spinner, 2) + homeLabel
	bar := skeletonCell(spinner, width-4)
	last := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}).
		Render("Last meeting: ") + skeletonCell(spinner, 18)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"})
	cellWidth := 22
	labels := []string{"Runs", "Run diff", "Avg score", "Avg total", "1-run games", "Largest margin"}
	var row1, row2 strings.Builder
	for i, label := range labels {
		cell := labelStyle.Render(label) + "\n" + skeletonCell(spinner, 10)
		col := lipgloss.NewStyle().Width(cellWidth).Render(cell)
		if i < 3 {
			row1.WriteString(col)
		} else {
			row2.WriteString(col)
		}
	}

	return header + "\n" + bar + "\n\n" + last + "\n\n" + row1.String() + "\n" + row2.String()
}
