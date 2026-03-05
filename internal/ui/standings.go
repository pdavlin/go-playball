package ui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pdavlin/go-playball/internal/api"
	"github.com/pdavlin/go-playball/internal/ui/anim"
)

var divisionSortOrder = map[string]int{
	"East":    0,
	"Central": 1,
	"West":    2,
}

// handleStandingsKeys handles keyboard input for standings view
// TODO: Add scroll offset tracking and scroll indicators when standings gets scroll support.
func (m Model) handleStandingsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return m, nil
}

// renderStandings renders the standings view
func (m Model) renderStandings() string {
	var b strings.Builder

	title := titleStyle.Render("MLB Standings")
	b.WriteString(title)
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		return b.String()
	}

	if len(m.standings) == 0 {
		b.WriteString(itemStyle.Render("Loading standings..."))
		return b.String()
	}

	var alDivisions, nlDivisions []api.DivisionStandings
	for _, div := range m.standings {
		if strings.Contains(div.Division.Name, "American League") {
			alDivisions = append(alDivisions, div)
		} else {
			nlDivisions = append(nlDivisions, div)
		}
	}
	sortDivisions(alDivisions)
	sortDivisions(nlDivisions)

	if m.width >= 120 {
		panelWidth := (m.width - 2) / 2
		var gridRows []string
		for i := 0; i < 3; i++ {
			var left, right string
			if i < len(alDivisions) {
				left = m.renderDivisionPanel(alDivisions[i], panelWidth)
			}
			if i < len(nlDivisions) {
				right = m.renderDivisionPanel(nlDivisions[i], panelWidth)
			}
			row := lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)
			gridRows = append(gridRows, row)
		}
		b.WriteString(lipgloss.JoinVertical(lipgloss.Left, gridRows...))
	} else {
		panelWidth := m.width
		for i := 0; i < 3; i++ {
			if i < len(alDivisions) {
				b.WriteString(m.renderDivisionPanel(alDivisions[i], panelWidth))
				b.WriteString("\n")
			}
			if i < len(nlDivisions) {
				b.WriteString(m.renderDivisionPanel(nlDivisions[i], panelWidth))
				b.WriteString("\n")
			}
		}
	}

	return b.String()
}

// renderDivisionPanel renders a single division as a bordered panel with a table
func (m Model) renderDivisionPanel(div api.DivisionStandings, panelWidth int) string {
	title := shortenDivisionName(div.Division.Name)
	textWidth := panelWidth - 4 // border (2) + padding (2)

	headers := []string{"Team", "W", "L", "PCT", "GB", "WCGB", "L10", "STRK"}
	widths := []int{0, 3, 3, 5, 5, 5, 5, 5}

	var rows [][]string
	for _, team := range div.TeamRecords {
		teamName := GetTeamShortName(team.Team.Name)
		if m.config.IsFavoriteTeam(team.Team.Name) {
			teamName = "* " + teamName
		}

		l10 := "-"
		for _, sr := range team.LastTenGames.SplitRecords {
			if sr.Type == "lastTen" {
				l10 = fmt.Sprintf("%d-%d", sr.Wins, sr.Losses)
				break
			}
		}
		streak := team.Streak.StreakCode
		if streak == "" {
			streak = "-"
		}
		wcgb := team.WildCardGamesBack
		if wcgb == "" {
			wcgb = "-"
		}

		rows = append(rows, []string{
			teamName,
			fmt.Sprintf("%d", team.Wins),
			fmt.Sprintf("%d", team.Losses),
			team.WinningPercentage,
			team.GamesBack,
			wcgb, l10, streak,
		})
	}

	// Build table manually for per-row favorite highlighting
	resolved := resolveWidths(headers, widths, rows, textWidth)
	headerLine := formatRow(headers, resolved)

	hdrStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#000000"}).
		Background(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#FFFFFF"})

	var lines []string
	lines = append(lines, hdrStyle.Render(headerLine))
	for i, row := range rows {
		line := formatRow(row, resolved)
		if m.config.IsFavoriteTeam(div.TeamRecords[i].Team.Name) {
			teamColors := GetTeamColors(div.TeamRecords[i].Team.Name)
			lines = append(lines, anim.BlendGradientBold(line, teamColors.Primary, teamColors.Secondary))
		} else {
			lines = append(lines, line)
		}
	}
	tableContent := strings.Join(lines, "\n")

	borderColor := lipgloss.AdaptiveColor{Light: "#CCCCCC", Dark: "#444444"}
	return renderStaticPanel(title, tableContent, panelWidth, borderColor)
}

// renderStaticPanel renders content in a bordered panel (no scroll/focus)
func renderStaticPanel(title string, content string, panelWidth int, borderColor lipgloss.TerminalColor) string {
	titleLine := lipgloss.NewStyle().Bold(true).Foreground(colorSecondary).Render(title)

	contentWidth := panelWidth - 2
	if contentWidth < 10 {
		contentWidth = 10
	}

	body := titleLine + "\n" + content

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(contentWidth).
		BorderForeground(borderColor).
		Render(body)
}

// sortDivisions sorts divisions by East, Central, West
func sortDivisions(divisions []api.DivisionStandings) {
	sort.SliceStable(divisions, func(i, j int) bool {
		iKey := extractSubDivision(divisions[i].Division.Name)
		jKey := extractSubDivision(divisions[j].Division.Name)
		return divisionSortOrder[iKey] < divisionSortOrder[jKey]
	})
}

// extractSubDivision extracts "East", "Central", or "West" from a division name
func extractSubDivision(name string) string {
	for key := range divisionSortOrder {
		if strings.Contains(name, key) {
			return key
		}
	}
	return name
}

// shortenDivisionName converts "American League East" to "AL East"
func shortenDivisionName(name string) string {
	name = strings.Replace(name, "American League", "AL", 1)
	name = strings.Replace(name, "National League", "NL", 1)
	return name
}
