package ui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pdavlin/go-playball/internal/api"
	"github.com/pdavlin/go-playball/internal/ui/anim"
)

var (
	battingHeaders = []string{"Batters", "AB", "R", "H", "RBI", "BB", "K", "AVG", "OPS"}
	battingWidths  = []int{0, 4, 4, 4, 4, 4, 4, 6, 6}

	pitchingHeaders = []string{"Pitchers", "IP", "H", "R", "ER", "BB", "K", "HR", "ERA"}
	pitchingWidths  = []int{0, 5, 4, 4, 4, 4, 4, 4, 6}
)

// minSideBySideWidth is the minimum terminal width for 2x2 grid layout.
// Below this, panels stack vertically instead.
const minSideBySideWidth = 120

// renderBoxScore composes the four box score panels in a 2x2 grid
// (away left, home right, batting top, pitching bottom) when the terminal
// is wide enough, falling back to vertical stacking when narrow.
func (m Model) renderBoxScore(game *api.Game) string {
	if game.LiveData == nil {
		return lipgloss.NewStyle().
			Width(m.width).
			Align(lipgloss.Center).
			Render("Box score unavailable")
	}

	boxscore := game.LiveData.Boxscore
	var players map[string]api.GameDataPlayer
	if game.GameData != nil {
		players = game.GameData.Players
	}

	awayName := game.Teams.Away.Team.Name
	homeName := game.Teams.Home.Team.Name
	if awayName == "" && game.GameData != nil {
		awayName = game.GameData.Teams.Away.Name
	}
	if homeName == "" && game.GameData != nil {
		homeName = game.GameData.Teams.Home.Name
	}
	awayColors := GetTeamColors(awayName)
	homeColors := GetTeamColors(homeName)
	awayAbbr := getTeamAbbreviation(awayName)
	homeAbbr := getTeamAbbreviation(homeName)

	// Build rows
	awayBatRows := buildBatterRows(boxscore.Teams.Away, players)
	homeBatRows := buildBatterRows(boxscore.Teams.Home, players)
	awayPitchRows := buildPitcherRows(boxscore.Teams.Away, players)
	homePitchRows := buildPitcherRows(boxscore.Teams.Home, players)

	// Header overhead: scoreboard(4) + linescore(5) + decisions(3) + helpbar(1) = 13
	headerHeight := 13
	availableHeight := m.height - headerHeight
	if availableHeight < 12 {
		availableHeight = 12
	}

	// Determine panel dimensions based on layout mode.
	// lipgloss Width = content width (includes padding, excludes border).
	// With Padding(0,1) and RoundedBorder: text area = Width - 2, outer = Width + 2.
	// So for desired outer panelWidth: Width = panelWidth - 2, text area = panelWidth - 4.
	sideBySide := m.width >= minSideBySideWidth

	var panelWidth int
	if sideBySide {
		panelWidth = (m.width - 2) / 2 // 2 chars gap between columns
	} else {
		panelWidth = m.width
	}
	textWidth := panelWidth - 4 // usable text area inside border + padding

	// Render tables constrained to text width
	awayBatTable := renderTable(battingHeaders, battingWidths, awayBatRows, textWidth)
	homeBatTable := renderTable(battingHeaders, battingWidths, homeBatRows, textWidth)
	awayPitchTable := renderTable(pitchingHeaders, pitchingWidths, awayPitchRows, textWidth)
	homePitchTable := renderTable(pitchingHeaders, pitchingWidths, homePitchRows, textWidth)

	if sideBySide {
		return m.renderBoxScore2x2(
			awayAbbr, homeAbbr,
			awayBatTable, homeBatTable, awayPitchTable, homePitchTable,
			awayColors, homeColors,
			availableHeight, panelWidth,
		)
	}

	return m.renderBoxScoreStacked(
		awayAbbr, homeAbbr,
		awayBatTable, homeBatTable, awayPitchTable, homePitchTable,
		awayColors, homeColors,
		availableHeight, panelWidth,
	)
}

// renderBoxScore2x2 renders the 2x2 grid layout matching the React app:
// [away batting] [home batting]
// [away pitching] [home pitching]
func (m Model) renderBoxScore2x2(
	awayAbbr, homeAbbr string,
	awayBat, homeBat, awayPitch, homePitch string,
	awayColors, homeColors TeamColors,
	availableHeight int, panelWidth int,
) string {
	// Each panel adds 2 border lines (top + bottom). Two rows = 4 border lines.
	rowHeight := (availableHeight - 4) / 2
	if rowHeight < 5 {
		rowHeight = 5
	}

	awayBatPanel := m.renderPanel(awayAbbr+" Batting", awayBat, 0, rowHeight, panelWidth, awayColors)
	homeBatPanel := m.renderPanel(homeAbbr+" Batting", homeBat, 1, rowHeight, panelWidth, homeColors)
	awayPitchPanel := m.renderPanel(awayAbbr+" Pitching", awayPitch, 2, rowHeight, panelWidth, awayColors)
	homePitchPanel := m.renderPanel(homeAbbr+" Pitching", homePitch, 3, rowHeight, panelWidth, homeColors)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, awayBatPanel, "  ", homeBatPanel)
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, awayPitchPanel, "  ", homePitchPanel)

	return lipgloss.JoinVertical(lipgloss.Left, topRow, bottomRow)
}

// renderBoxScoreStacked renders all four panels in a single vertical stack
// for narrow terminals.
func (m Model) renderBoxScoreStacked(
	awayAbbr, homeAbbr string,
	awayBat, homeBat, awayPitch, homePitch string,
	awayColors, homeColors TeamColors,
	availableHeight int, panelWidth int,
) string {
	// Each panel adds 2 border lines. Four panels = 8 border lines.
	panelHeight := (availableHeight - 8) / 4
	if panelHeight < 3 {
		panelHeight = 3
	}

	panels := []string{
		m.renderPanel(awayAbbr+" Batting", awayBat, 0, panelHeight, panelWidth, awayColors),
		m.renderPanel(homeAbbr+" Batting", homeBat, 1, panelHeight, panelWidth, homeColors),
		m.renderPanel(awayAbbr+" Pitching", awayPitch, 2, panelHeight, panelWidth, awayColors),
		m.renderPanel(homeAbbr+" Pitching", homePitch, 3, panelHeight, panelWidth, homeColors),
	}

	return lipgloss.JoinVertical(lipgloss.Left, panels...)
}

// renderPanel wraps content in a bordered panel with scroll support.
// panelWidth controls the outer width of the panel (including border).
func (m Model) renderPanel(title string, content string, panelIdx int, height int, panelWidth int, teamColors TeamColors) string {
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")

	// The usable inner height accounts for the title line
	innerHeight := height - 1
	if innerHeight < 1 {
		innerHeight = 1
	}

	offset := m.panelScrollOffsets[panelIdx]
	if offset < 0 {
		offset = 0
	}

	// Determine which indicators are needed and adjust height
	hasUp := offset > 0
	hasDown := len(lines) > offset+innerHeight
	adjustedHeight := innerHeight
	if hasUp {
		adjustedHeight--
	}
	if hasDown {
		adjustedHeight--
	}
	if adjustedHeight < 1 {
		adjustedHeight = 1
	}

	// Clamp scroll offset with adjusted height
	maxOffset := len(lines) - adjustedHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}

	// Re-check indicators after clamping
	hasUp = offset > 0
	end := offset + adjustedHeight
	if end > len(lines) {
		end = len(lines)
	}
	hasDown = end < len(lines)

	visible := lines[offset:end]

	// Text area width for centering indicators
	textWidth := panelWidth - 4

	// Build content with title
	var b strings.Builder
	if panelIdx == m.focusedPanel {
		b.WriteString(anim.BlendGradientBold(title, teamColors.Primary, teamColors.Secondary))
	} else {
		titleStyle := lipgloss.NewStyle().Bold(true)
		b.WriteString(titleStyle.Render(title))
	}
	b.WriteString("\n")

	if hasUp {
		b.WriteString(anim.ScrollIndicator(anim.ScrollUp, offset, textWidth, teamColors.Primary, teamColors.Secondary))
		b.WriteString("\n")
	}

	b.WriteString(strings.Join(visible, "\n"))

	if hasDown {
		remaining := len(lines) - end
		b.WriteString("\n")
		b.WriteString(anim.ScrollIndicator(anim.ScrollDown, remaining, textWidth, teamColors.Primary, teamColors.Secondary))
	}

	// lipgloss Width/Height = content area (includes padding, excludes border).
	// Outer = Width/Height + 2 (border). So Width = panelWidth - 2.
	contentWidth := panelWidth - 2
	if contentWidth < 10 {
		contentWidth = 10
	}

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(contentWidth).
		Height(height)

	if panelIdx == m.focusedPanel {
		midColor := anim.GradientRamp(teamColors.Primary, teamColors.Secondary, 3)[1]
		borderStyle = borderStyle.BorderForeground(lipgloss.Color(anim.ColorToHex(midColor)))
	} else {
		borderStyle = borderStyle.BorderForeground(lipgloss.AdaptiveColor{Light: "#CCCCCC", Dark: "#444444"})
	}

	return borderStyle.Render(b.String())
}

// buildBatterRows builds table rows for a team's batting stats.
func buildBatterRows(team api.BoxscoreTeam, players map[string]api.GameDataPlayer) [][]string {
	type batterEntry struct {
		order  int
		player api.BoxscorePlayer
		key    string
	}

	var batters []batterEntry
	for key, p := range team.Players {
		if p.BattingOrder == "" {
			continue
		}
		order, err := strconv.Atoi(p.BattingOrder)
		if err != nil {
			continue
		}
		batters = append(batters, batterEntry{order: order, player: p, key: key})
	}

	sort.Slice(batters, func(i, j int) bool {
		return batters[i].order < batters[j].order
	})

	var rows [][]string
	for _, entry := range batters {
		p := entry.player
		name := getPlayerName(p, players)
		pos := getPositionString(p)

		if pos != "" {
			name = name + " (" + pos + ")"
		}

		ab, r, h, rbi, bb, k := "0", "0", "0", "0", "0", "0"
		if p.Stats.Batting != nil {
			ab = strconv.Itoa(p.Stats.Batting.AtBats)
			r = strconv.Itoa(p.Stats.Batting.Runs)
			h = strconv.Itoa(p.Stats.Batting.Hits)
			rbi = strconv.Itoa(p.Stats.Batting.RBI)
			bb = strconv.Itoa(p.Stats.Batting.BaseOnBalls)
			k = strconv.Itoa(p.Stats.Batting.StrikeOuts)
		}

		avg, ops := "---", "---"
		if p.SeasonStats != nil && p.SeasonStats.Batting != nil {
			avg = p.SeasonStats.Batting.Avg
			ops = p.SeasonStats.Batting.OPS
		}

		rows = append(rows, []string{name, ab, r, h, rbi, bb, k, avg, ops})
	}

	// Totals row
	ts := team.TeamStats.Batting
	rows = append(rows, []string{
		"Totals",
		strconv.Itoa(ts.AtBats),
		strconv.Itoa(ts.Runs),
		strconv.Itoa(ts.Hits),
		strconv.Itoa(ts.RBI),
		strconv.Itoa(ts.BaseOnBalls),
		strconv.Itoa(ts.StrikeOuts),
		"", "",
	})

	return rows
}

// buildPitcherRows builds table rows for a team's pitching stats.
func buildPitcherRows(team api.BoxscoreTeam, players map[string]api.GameDataPlayer) [][]string {
	var rows [][]string

	for _, pid := range team.Pitchers {
		key := fmt.Sprintf("ID%d", pid)
		p, ok := team.Players[key]
		if !ok {
			continue
		}

		name := getPlayerName(p, players)

		// Add pitching note (W, L, S, H, etc.)
		if p.Stats.Pitching != nil && p.Stats.Pitching.Note != "" {
			name = name + " (" + p.Stats.Pitching.Note + ")"
		}

		ip, h, r, er, bb, k, hr := "0.0", "0", "0", "0", "0", "0", "0"
		if p.Stats.Pitching != nil {
			ip = p.Stats.Pitching.InningsPitched
			h = strconv.Itoa(p.Stats.Pitching.Hits)
			r = strconv.Itoa(p.Stats.Pitching.Runs)
			er = strconv.Itoa(p.Stats.Pitching.EarnedRuns)
			bb = strconv.Itoa(p.Stats.Pitching.BaseOnBalls)
			k = strconv.Itoa(p.Stats.Pitching.StrikeOuts)
			hr = strconv.Itoa(p.Stats.Pitching.HomeRuns)
		}

		era := "---"
		if p.SeasonStats != nil && p.SeasonStats.Pitching != nil {
			era = p.SeasonStats.Pitching.Era
		}

		rows = append(rows, []string{name, ip, h, r, er, bb, k, hr, era})
	}

	// Totals row
	ts := team.TeamStats.Pitching
	rows = append(rows, []string{
		"Totals",
		ts.InningsPitched,
		strconv.Itoa(ts.Hits),
		strconv.Itoa(ts.Runs),
		strconv.Itoa(ts.EarnedRuns),
		strconv.Itoa(ts.BaseOnBalls),
		strconv.Itoa(ts.StrikeOuts),
		strconv.Itoa(ts.HomeRuns),
		"",
	})

	return rows
}

// getPlayerName returns the boxscore display name for a player.
func getPlayerName(player api.BoxscorePlayer, players map[string]api.GameDataPlayer) string {
	key := fmt.Sprintf("ID%d", player.Person.ID)
	if gdp, ok := players[key]; ok && gdp.BoxscoreName != "" {
		return gdp.BoxscoreName
	}
	return player.Person.FullName
}

// getPositionString builds a position abbreviation string.
func getPositionString(player api.BoxscorePlayer) string {
	if len(player.AllPositions) > 0 {
		abbrs := make([]string, len(player.AllPositions))
		for i, pos := range player.AllPositions {
			abbrs[i] = pos.Abbreviation
		}
		return strings.Join(abbrs, "-")
	}
	if player.Position.Abbreviation != "" {
		return player.Position.Abbreviation
	}
	return ""
}
