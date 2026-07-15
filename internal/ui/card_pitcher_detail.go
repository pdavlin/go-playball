package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pdavlin/go-playball/internal/api"
	"github.com/pdavlin/go-playball/internal/ui/anim"
)

// pitcherDetailSideBySideWidth is the minimum total card width at which
// the two pitcher columns render side-by-side instead of stacked.
const pitcherDetailSideBySideWidth = 100

// Skeleton row counts while loading. Picked to match a typical starter:
// most pitchers throw 4-6 distinct pitches and we cap recent starts at 5.
const (
	skeletonArsenalRows = 5
	skeletonStartsRows  = 5
)

// renderPitcherDetailCard renders the Pitchers tab body. When the fetch
// is in-flight (data nil or !loaded), it renders the table shape with
// spinner-filled skeleton cells instead of a single "Loading..." string.
func renderPitcherDetailCard(game *api.Game, data *pregameGameData, spinner *anim.Spinner, width, height int) string {
	if game == nil || game.GameData == nil {
		return padToHeight(itemStyle.Render("Game data unavailable"), height)
	}

	awayName := game.GameData.Teams.Away.Name
	homeName := game.GameData.Teams.Home.Name
	awayProb := game.GameData.ProbablePitchers.Away
	homeProb := game.GameData.ProbablePitchers.Home
	awayColors := GetTeamColors(awayName)
	homeColors := GetTeamColors(homeName)

	loading := data == nil || data.pitcherDetail == nil || !data.pitcherDetail.loaded

	var awayCol, homeCol string
	if loading {
		awayCol = renderPitcherColumnSkeleton(awayProb, awayColors, spinner)
		homeCol = renderPitcherColumnSkeleton(homeProb, homeColors, spinner)
	} else {
		payload := data.pitcherDetail
		awayCol = renderPitcherColumn(
			awayProb, awayColors,
			payload.awayArsenal, payload.awayLog, payload.awayErr,
		)
		homeCol = renderPitcherColumn(
			homeProb, homeColors,
			payload.homeArsenal, payload.homeLog, payload.homeErr,
		)
	}

	// leftGutter pushes the away column away from the terminal edge so
	// the table doesn't read as flush against the border.
	const leftGutter = 2
	usable := width - leftGutter

	var body string
	if usable >= pitcherDetailSideBySideWidth {
		colWidth := (usable - 4) / 2
		if colWidth < 30 {
			colWidth = 30
		}
		left := lipgloss.NewStyle().Width(colWidth).Render(awayCol)
		gap := "  "
		right := lipgloss.NewStyle().Width(colWidth).Render(homeCol)
		body = lipgloss.JoinHorizontal(lipgloss.Top, left, gap, right)
	} else {
		body = lipgloss.JoinVertical(lipgloss.Left, awayCol, "", homeCol)
	}
	body = lipgloss.NewStyle().PaddingLeft(leftGutter).Render(body)
	return padToHeight(body, height)
}

// renderPitcherColumn renders one pitcher's section: header + arsenal
// table + recent-starts table. Falls back gracefully on TBD, error,
// and empty data.
func renderPitcherColumn(
	prob api.ProbablePitcher,
	colors TeamColors,
	arsenal []api.ArsenalPitch,
	log []api.GameLogStart,
	fetchErr error,
) string {
	var b strings.Builder

	nameStyle := lipgloss.NewStyle().Foreground(colors.Primary).Bold(true)

	if prob.ID == 0 {
		b.WriteString(nameStyle.Render("Probable pitcher TBD"))
		return b.String()
	}

	b.WriteString(nameStyle.Render(prob.FullName))
	b.WriteString("\n\n")

	if fetchErr != nil {
		b.WriteString(itemStyle.Render("Data unavailable"))
		return b.String()
	}

	b.WriteString(headerStyle.Render("Arsenal"))
	b.WriteString("\n")
	b.WriteString(renderArsenalTable(arsenal))
	b.WriteString("\n\n")

	b.WriteString(headerStyle.Render("Recent starts"))
	b.WriteString("\n")
	b.WriteString(renderRecentStartsTable(log))

	return b.String()
}

func renderArsenalTable(rows []api.ArsenalPitch) string {
	if len(rows) == 0 {
		return itemStyle.Render("Arsenal unavailable")
	}
	sorted := make([]api.ArsenalPitch, len(rows))
	copy(sorted, rows)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].UsagePct > sorted[j].UsagePct
	})

	headerRow := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}).
		Bold(true).
		Render(fmt.Sprintf("%-18s %6s %5s", "Pitch", "Use%", "Velo"))

	var b strings.Builder
	b.WriteString(headerRow)
	b.WriteString("\n")
	for _, r := range sorted {
		veloStr := "    -"
		if r.AvgVelocity > 0 {
			veloStr = fmt.Sprintf("%5.1f", r.AvgVelocity)
		}
		name := r.Description
		if name == "" {
			name = r.Code
		}
		line := fmt.Sprintf("%-18s %5.1f%% %5s",
			truncatePitchName(name, 18), r.UsagePct, veloStr)
		b.WriteString(line)
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderRecentStartsTable(rows []api.GameLogStart) string {
	if len(rows) == 0 {
		return itemStyle.Render("No prior starts this season")
	}
	headerRow := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}).
		Bold(true).
		Render(fmt.Sprintf("%-10s %-4s %4s %2s %2s %2s %2s %4s",
			"Date", "Opp", "IP", "ER", "K", "BB", "HR", "P"))

	var b strings.Builder
	b.WriteString(headerRow)
	b.WriteString("\n")
	for _, r := range rows {
		opp := getTeamAbbreviation(r.OpponentName)
		line := fmt.Sprintf("%-10s %-4s %4s %2d %2d %2d %2d %4d",
			r.Date, opp, r.InningsPitched,
			r.EarnedRuns, r.Strikeouts, r.Walks, r.HomeRuns, r.Pitches)
		b.WriteString(line)
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// renderPitcherColumnSkeleton renders one pitcher's column shape with
// spinner-filled cells in place of real values. TBD probable pitchers
// short-circuit to the same fallback as the loaded path; everything else
// gets the full table skeleton.
func renderPitcherColumnSkeleton(
	prob api.ProbablePitcher,
	colors TeamColors,
	spinner *anim.Spinner,
) string {
	var b strings.Builder

	nameStyle := lipgloss.NewStyle().Foreground(colors.Primary).Bold(true)

	if prob.ID == 0 {
		b.WriteString(nameStyle.Render("Probable pitcher TBD"))
		return b.String()
	}

	b.WriteString(nameStyle.Render(prob.FullName))
	b.WriteString("\n\n")

	b.WriteString(headerStyle.Render("Arsenal"))
	b.WriteString("\n")
	b.WriteString(renderArsenalSkeleton(spinner))
	b.WriteString("\n\n")

	b.WriteString(headerStyle.Render("Recent starts"))
	b.WriteString("\n")
	b.WriteString(renderRecentStartsSkeleton(spinner))

	return b.String()
}

// renderArsenalSkeleton renders the Arsenal table header plus N spinner
// rows. Spaces between columns match the loaded layout so the row
// alignment doesn't shift when real data lands.
func renderArsenalSkeleton(spinner *anim.Spinner) string {
	headerRow := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}).
		Bold(true).
		Render(fmt.Sprintf("%-18s %6s %5s", "Pitch", "Use%", "Velo"))

	var b strings.Builder
	b.WriteString(headerRow)
	b.WriteString("\n")
	for i := 0; i < skeletonArsenalRows; i++ {
		row := skeletonCell(spinner, 18) + " " +
			skeletonCell(spinner, 6) + " " +
			skeletonCell(spinner, 5)
		b.WriteString(row)
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// renderRecentStartsSkeleton mirrors renderRecentStartsTable's column
// widths but fills cells with the spinner.
func renderRecentStartsSkeleton(spinner *anim.Spinner) string {
	headerRow := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}).
		Bold(true).
		Render(fmt.Sprintf("%-10s %-4s %4s %2s %2s %2s %2s %4s",
			"Date", "Opp", "IP", "ER", "K", "BB", "HR", "P"))

	var b strings.Builder
	b.WriteString(headerRow)
	b.WriteString("\n")
	for i := 0; i < skeletonStartsRows; i++ {
		row := skeletonCell(spinner, 10) + " " +
			skeletonCell(spinner, 4) + " " +
			skeletonCell(spinner, 4) + " " +
			skeletonCell(spinner, 2) + " " +
			skeletonCell(spinner, 2) + " " +
			skeletonCell(spinner, 2) + " " +
			skeletonCell(spinner, 2) + " " +
			skeletonCell(spinner, 4)
		b.WriteString(row)
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// skeletonCell returns n visual cells of spinner output, padded to width
// when the spinner is nil or shorter than requested.
func skeletonCell(spinner *anim.Spinner, n int) string {
	if spinner == nil {
		return strings.Repeat(" ", n)
	}
	s := spinner.TrailerView(n)
	return lipgloss.NewStyle().Width(n).Render(s)
}
