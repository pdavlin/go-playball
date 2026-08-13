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
	// Stat columns are sized to their content (e.g. "RBI", ".238", "1.000")
	// rather than padded generously, so the name column keeps as much of
	// the reclaimed panel width as possible (see F5).
	battingWidths = []int{0, 3, 3, 3, 3, 3, 3, 5, 5}

	pitchingHeaders = []string{"Pitchers", "IP", "H", "R", "ER", "BB", "K", "HR", "ERA"}
	pitchingWidths  = []int{0, 4, 3, 3, 3, 3, 3, 3, 5}
)

// minSideBySideWidth is the minimum terminal width for 2x2 grid layout.
// Below this, panels stack vertically instead.
const minSideBySideWidth = 120

// renderBoxScore composes the four box score panels in a 2x2 grid
// (away left, home right, batting top, pitching bottom) when the terminal
// is wide enough, falling back to vertical stacking when narrow.
// availableHeight is the number of screen lines the caller has left for
// the panels.
func (m Model) renderBoxScore(game *api.Game, availableHeight int) string {
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

	// Render tables constrained to text width. Away/home pairs share column
	// geometry (via renderTablePair) so headers and stat columns land at the
	// same x position in both panels, even when one side's names are longer.
	awayBatTable, homeBatTable := renderTablePair(battingHeaders, battingWidths, awayBatRows, homeBatRows, textWidth)
	awayPitchTable, homePitchTable := renderTablePair(pitchingHeaders, pitchingWidths, awayPitchRows, homePitchRows, textWidth)

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

// collapsedPanelHeight is the content height of an unfocused panel in
// the stacked layout: its title, one table row and the overflow
// indicator.
const collapsedPanelHeight = 3

// renderBoxScoreStacked renders all four panels in a single vertical
// stack for narrow terminals, as an accordion: the panel selected with
// 1-4 takes the height its table needs (up to what is left), the other
// three collapse to their title, one row and an overflow indicator (F10).
func (m Model) renderBoxScoreStacked(
	awayAbbr, homeAbbr string,
	awayBat, homeBat, awayPitch, homePitch string,
	awayColors, homeColors TeamColors,
	availableHeight int, panelWidth int,
) string {
	contents := []string{awayBat, homeBat, awayPitch, homePitch}
	titles := []string{
		awayAbbr + " Batting", homeAbbr + " Batting",
		awayAbbr + " Pitching", homeAbbr + " Pitching",
	}
	colors := []TeamColors{awayColors, homeColors, awayColors, homeColors}

	var contentLines [4]int
	for i, content := range contents {
		contentLines[i] = lipgloss.Height(content)
	}
	heights := stackedPanelHeights(availableHeight, contentLines, m.focusedPanel)

	panels := make([]string, len(contents))
	for i, content := range contents {
		panels[i] = m.renderPanel(titles[i], content, i, heights[i], panelWidth, colors[i])
	}

	return lipgloss.JoinVertical(lipgloss.Left, panels...)
}

// stackedPanelHeights budgets content heights for the four stacked
// panels. Each panel also draws 2 border lines, so the stack occupies
// sum(heights) + 8 screen lines. The focused panel absorbs everything
// the collapsed panels do not need, capped at the height its own table
// wants; slack from a short focused table goes back to the collapsed
// panels. When there is not enough room for an accordion the four
// panels split the budget evenly, as before.
func stackedPanelHeights(availableHeight int, contentLines [4]int, focused int) [4]int {
	var heights [4]int
	if focused < 0 || focused >= len(heights) {
		focused = 0
	}

	budget := availableHeight - 8 // border lines

	// The focused panel needs its title plus a couple more rows than a
	// collapsed one, or the accordion buys nothing over an even split.
	focusedHeight := budget - 3*collapsedPanelHeight
	if focusedHeight < collapsedPanelHeight+2 {
		even := budget / 4
		if even < 3 {
			even = 3
		}
		for i := range heights {
			heights[i] = even
		}
		return heights
	}

	// A panel never needs to be taller than its title plus its rows.
	var want [4]int
	for i, n := range contentLines {
		want[i] = n + 1
	}
	if want[focused] < focusedHeight {
		focusedHeight = want[focused]
	}

	used := focusedHeight
	for i := range heights {
		heights[i] = collapsedPanelHeight
		if i != focused {
			used += collapsedPanelHeight
		}
	}
	heights[focused] = focusedHeight

	// Hand slack from a short focused table back to the other panels so
	// the stack fills the screen instead of trailing off into blanks.
	for slack := budget - used; slack > 0; {
		grew := false
		for i := range heights {
			if slack == 0 {
				break
			}
			if i == focused || heights[i] >= want[i] {
				continue
			}
			heights[i]++
			slack--
			grew = true
		}
		if !grew {
			break
		}
	}

	return heights
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

	// Determine which indicators are needed and adjust height. The
	// indicators must fit inside the panel, so a collapsed panel can end
	// up showing no rows at all rather than overflowing its box.
	hasUp := offset > 0
	hasDown := len(lines) > offset+innerHeight
	adjustedHeight := innerHeight
	if hasUp {
		adjustedHeight--
	}
	if hasDown {
		adjustedHeight--
	}
	if adjustedHeight < 0 {
		adjustedHeight = 0
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

	// Build the body as an explicit line list: title, optional
	// indicators and the visible rows never exceed height lines.
	// Only the focused panel's title is bold (gradient) — the weight
	// difference is the non-color selection cue.
	titleLine := title
	if panelIdx == m.focusedPanel {
		titleLine = anim.BlendGradientBold(title, teamColors.Primary, teamColors.Secondary)
	}

	body := []string{titleLine}
	if hasUp {
		body = append(body, anim.ScrollIndicator(anim.ScrollUp, offset, textWidth, teamColors.Primary, teamColors.Secondary))
	}
	body = append(body, visible...)
	if hasDown {
		body = append(body, anim.ScrollIndicator(anim.ScrollDown, len(lines)-end, textWidth, teamColors.Primary, teamColors.Secondary))
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
		borderStyle = borderStyle.BorderForeground(lipgloss.AdaptiveColor{Light: "#AAAAAA", Dark: "#666666"})
	}

	return borderStyle.Render(strings.Join(body, "\n"))
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

		// Add pitching note (W, L, S, H, etc.). The API's Note field
		// sometimes already includes its own parens, so strip any
		// existing wrapping before re-adding ours to avoid "((".
		if p.Stats.Pitching != nil && p.Stats.Pitching.Note != "" {
			note := strings.Trim(p.Stats.Pitching.Note, "()")
			name = name + " (" + note + ")"
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
