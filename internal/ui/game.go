package ui

import (
	"fmt"
	"math"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pdavlin/go-playball/internal/api"
	"github.com/pdavlin/go-playball/internal/ui/anim"
)

// handleGameKeys handles keyboard input for game view
func (m Model) handleGameKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.currentGame == nil {
		return m, nil
	}

	gameState := m.currentGame.Status.AbstractGameState
	if gameState == "" && m.currentGame.GameData != nil {
		gameState = m.currentGame.GameData.Status.AbstractGameState
	}

	// View switching keys (work in all subviews for both live and final)
	switch msg.String() {
	case "b":
		m.gameSubview = BoxScoreSubview
		m.focusedPanel = 0
		m.panelScrollOffsets = [4]int{}
		m.gameScrollOffset = 0
		return m, nil
	case "a":
		m.gameSubview = AllPlaysSubview
		m.focusedPanel = 0
		m.panelScrollOffsets = [4]int{}
		m.gameScrollOffset = 0
		return m, nil
	case "p":
		m.gameSubview = ScoringPlaysSubview
		m.focusedPanel = 0
		m.panelScrollOffsets = [4]int{}
		m.gameScrollOffset = 0
		return m, nil
	case "g":
		m.gameSubview = GameStatusSubview
		m.gameScrollOffset = 0
		return m, nil
	}

	// Subview-specific keys
	switch m.gameSubview {
	case BoxScoreSubview:
		return m.handleBoxScoreKeys(msg)
	case AllPlaysSubview, ScoringPlaysSubview:
		return m.handlePlaysKeys(msg)
	case GameStatusSubview:
		return m.handleGameStatusKeys(msg)
	}

	return m, nil
}

// handleBoxScoreKeys handles keys for box score panel navigation
func (m Model) handleBoxScoreKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "1":
		m.focusedPanel = 0
	case "2":
		m.focusedPanel = 1
	case "3":
		m.focusedPanel = 2
	case "4":
		m.focusedPanel = 3
	case "up", "k":
		if m.panelScrollOffsets[m.focusedPanel] > 0 {
			m.panelScrollOffsets[m.focusedPanel]--
		}
	case "down", "j":
		m.panelScrollOffsets[m.focusedPanel]++
	case "g":
		m.panelScrollOffsets[m.focusedPanel] = 0
	case "G":
		m.panelScrollOffsets[m.focusedPanel] = 9999
	}
	return m, nil
}

// handlePlaysKeys handles keys for all-plays and scoring-plays scroll
func (m Model) handlePlaysKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.gameScrollOffset > 0 {
			m.gameScrollOffset--
		}
	case "down", "j":
		m.gameScrollOffset++
	case "g":
		m.gameScrollOffset = 0
	case "G":
		m.gameScrollOffset = 9999
	}
	return m, nil
}

// handleGameStatusKeys handles keys for the default live game status view
func (m Model) handleGameStatusKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.gameScrollOffset > 0 {
			m.gameScrollOffset--
		}
	case "down", "j":
		m.gameScrollOffset++
	case "g":
		m.gameScrollOffset = 0
	case "G":
		m.gameScrollOffset = 9999
	}
	return m, nil
}

// renderGame renders the live game view
func (m Model) renderGame() string {
	var b strings.Builder

	// Show error if present
	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error loading game: %v", m.err)))
		return b.String()
	}

	if m.currentGame == nil {
		if m.loading {
			b.WriteString(itemStyle.Render("Loading game..."))
		} else {
			b.WriteString(errorStyle.Render("No game data available"))
		}
		return b.String()
	}

	game := m.currentGame

	// Determine game state - check GameData.Status if game.Status is empty
	gameState := game.Status.AbstractGameState
	if gameState == "" && game.GameData != nil {
		gameState = game.GameData.Status.AbstractGameState
	}

	// Render based on game state
	// Preview and Live get a title header; Final has the scoreboard instead
	switch gameState {
	case "Preview":
		title := fmt.Sprintf("%s @ %s", GetTeamShortName(game.Teams.Away.Team.Name), GetTeamShortName(game.Teams.Home.Team.Name))
		b.WriteString(titleStyle.Render(title))
		b.WriteString("\n\n")
		b.WriteString(m.renderPreviewGame(game))
	case "Live":
		title := fmt.Sprintf("%s @ %s", GetTeamShortName(game.Teams.Away.Team.Name), GetTeamShortName(game.Teams.Home.Team.Name))
		b.WriteString(titleStyle.Render(title))
		b.WriteString("\n\n")
		b.WriteString(m.renderLiveGame(game))
	case "Final":
		b.WriteString(m.renderFinalGame(game))
	default:
		// Debug: show what we got
		detailedState := game.Status.DetailedState
		if detailedState == "" && game.GameData != nil {
			detailedState = game.GameData.Status.DetailedState
		}
		b.WriteString(itemStyle.Render(fmt.Sprintf("Game State: '%s', Detailed: '%s'", gameState, detailedState)))
	}

	return b.String()
}

// renderPreviewGame renders a preview (upcoming) game in three-column layout
func (m Model) renderPreviewGame(game *api.Game) string {
	if game.GameData == nil {
		return itemStyle.Render("Game data unavailable")
	}

	// Get team names and colors
	awayName := GetTeamShortName(game.GameData.Teams.Away.Name)
	homeName := GetTeamShortName(game.GameData.Teams.Home.Name)
	awayColors := GetTeamColors(game.GameData.Teams.Away.Name)
	homeColors := GetTeamColors(game.GameData.Teams.Home.Name)

	// Get series records (or league records as fallback)
	awayRecord := ""
	homeRecord := ""
	if game.GameData.Teams.Away.Record != nil {
		awayRecord = fmt.Sprintf("(%d-%d)",
			game.GameData.Teams.Away.Record.Wins,
			game.GameData.Teams.Away.Record.Losses)
	} else {
		awayRecord = fmt.Sprintf("(%d-%d)",
			game.GameData.Teams.Away.LeagueRecord.Wins,
			game.GameData.Teams.Away.LeagueRecord.Losses)
	}
	if game.GameData.Teams.Home.Record != nil {
		homeRecord = fmt.Sprintf("(%d-%d)",
			game.GameData.Teams.Home.Record.Wins,
			game.GameData.Teams.Home.Record.Losses)
	} else {
		homeRecord = fmt.Sprintf("(%d-%d)",
			game.GameData.Teams.Home.LeagueRecord.Wins,
			game.GameData.Teams.Home.LeagueRecord.Losses)
	}

	// Format away team column
	awayLines := []string{
		lipgloss.NewStyle().Foreground(awayColors.Primary).Bold(true).Render(awayName),
		lipgloss.NewStyle().Foreground(awayColors.Secondary).Render(awayRecord),
	}

	// Add away pitcher info from boxscore
	if game.LiveData != nil && game.GameData.ProbablePitchers.Away.ID != 0 {
		pitcherKey := fmt.Sprintf("ID%d", game.GameData.ProbablePitchers.Away.ID)
		if pitcher, ok := game.LiveData.Boxscore.Teams.Away.Players[pitcherKey]; ok {
			awayLines = append(awayLines, "")
			pitcherName := pitcher.Person.FullName
			if pitcher.JerseyNumber != "" {
				pitcherName = fmt.Sprintf("%s, #%s", pitcherName, pitcher.JerseyNumber)
			}
			awayLines = append(awayLines, lipgloss.NewStyle().Foreground(awayColors.Primary).Render(pitcherName))

			if pitcher.SeasonStats != nil && pitcher.SeasonStats.Pitching != nil {
				stats := pitcher.SeasonStats.Pitching
				awayLines = append(awayLines,
					lipgloss.NewStyle().Foreground(awayColors.Secondary).Render(fmt.Sprintf("%d-%d", stats.Wins, stats.Losses)),
					lipgloss.NewStyle().Foreground(awayColors.Secondary).Render(fmt.Sprintf("%s ERA %d K", stats.Era, stats.StrikeOuts)),
				)
			}
		}
	}

	// Format home team column
	homeLines := []string{
		lipgloss.NewStyle().Foreground(homeColors.Primary).Bold(true).Render(homeName),
		lipgloss.NewStyle().Foreground(homeColors.Secondary).Render(homeRecord),
	}

	// Add home pitcher info from boxscore
	if game.LiveData != nil && game.GameData.ProbablePitchers.Home.ID != 0 {
		pitcherKey := fmt.Sprintf("ID%d", game.GameData.ProbablePitchers.Home.ID)
		if pitcher, ok := game.LiveData.Boxscore.Teams.Home.Players[pitcherKey]; ok {
			homeLines = append(homeLines, "")
			pitcherName := pitcher.Person.FullName
			if pitcher.JerseyNumber != "" {
				pitcherName = fmt.Sprintf("%s, #%s", pitcherName, pitcher.JerseyNumber)
			}
			homeLines = append(homeLines, lipgloss.NewStyle().Foreground(homeColors.Primary).Render(pitcherName))

			if pitcher.SeasonStats != nil && pitcher.SeasonStats.Pitching != nil {
				stats := pitcher.SeasonStats.Pitching
				homeLines = append(homeLines,
					lipgloss.NewStyle().Foreground(homeColors.Secondary).Render(fmt.Sprintf("%d-%d", stats.Wins, stats.Losses)),
					lipgloss.NewStyle().Foreground(homeColors.Secondary).Render(fmt.Sprintf("%s ERA %d K", stats.Era, stats.StrikeOuts)),
				)
			}
		}
	}

	// Format center column with game info
	// Use GameData.Datetime.DateTime for accurate start time
	startTime := ""
	if !game.GameData.Datetime.DateTime.IsZero() {
		if game.GameData.Status.StartTimeTBD {
			startTime = "Start time TBD"
		} else {
			startTime = game.GameData.Datetime.DateTime.Local().Format("January 2, 2006 3:04 PM MST")
		}
	} else if !game.GameDate.IsZero() {
		startTime = game.GameDate.Local().Format("January 2, 2006 3:04 PM MST")
	}

	centerLines := []string{
		"",
		lipgloss.NewStyle().Bold(true).Render("vs."),
		"",
		startTime,
		game.GameData.Venue.Name,
	}
	// Add location if available
	if game.GameData.Venue.Location.City != "" {
		location := fmt.Sprintf("%s, %s", game.GameData.Venue.Location.City, game.GameData.Venue.Location.StateAbbrev)
		centerLines = append(centerLines, location)
	}

	// Create three columns
	colWidth := (m.width - 4) / 3
	if colWidth < 20 {
		colWidth = 20
	}

	awayCol := lipgloss.NewStyle().
		Width(colWidth).
		Align(lipgloss.Center).
		Render(strings.Join(awayLines, "\n"))

	centerCol := lipgloss.NewStyle().
		Width(colWidth).
		Align(lipgloss.Center).
		Render(strings.Join(centerLines, "\n"))

	homeCol := lipgloss.NewStyle().
		Width(colWidth).
		Align(lipgloss.Center).
		Render(strings.Join(homeLines, "\n"))

	// Join columns horizontally
	preview := lipgloss.JoinHorizontal(lipgloss.Top, awayCol, centerCol, homeCol)

	// Center on screen with padding
	centered := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		MarginTop(5).
		Render(preview)

	return centered
}

// renderLiveGame dispatches to the appropriate subview for a live game
func (m Model) renderLiveGame(game *api.Game) string {
	if game.LiveData == nil {
		return itemStyle.Render("Game data unavailable")
	}

	switch m.gameSubview {
	case BoxScoreSubview:
		header := m.renderCompactGameSituation(game)
		separator := lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#CCCCCC", Dark: "#444444"}).
			Render(strings.Repeat("─", m.width))
		return lipgloss.JoinVertical(lipgloss.Left, header, separator, m.renderBoxScore(game))
	case AllPlaysSubview:
		header := m.renderCompactGameSituation(game)
		separator := lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#CCCCCC", Dark: "#444444"}).
			Render(strings.Repeat("─", m.width))
		availableHeight := m.height - strings.Count(header, "\n") - 3
		return lipgloss.JoinVertical(lipgloss.Left, header, separator,
			m.renderPlays(game, availableHeight, m.width, false, true))
	case ScoringPlaysSubview:
		header := m.renderCompactGameSituation(game)
		separator := lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#CCCCCC", Dark: "#444444"}).
			Render(strings.Repeat("─", m.width))
		availableHeight := m.height - strings.Count(header, "\n") - 3
		return lipgloss.JoinVertical(lipgloss.Left, header, separator,
			m.renderPlays(game, availableHeight, m.width, true, true))
	default:
		return m.renderLiveGameStatus(game)
	}
}

// renderLiveGameStatus renders the default live game layout (matchup, at-bat, plays)
func (m Model) renderLiveGameStatus(game *api.Game) string {
	// Top row: Inning, Count, Bases, Line Score (all on one line)
	topRow := m.renderCompactGameSituation(game)

	// Separator line
	separator := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#CCCCCC", Dark: "#444444"}).
		Render(strings.Repeat("─", m.width))

	// Bottom section: Two columns
	// Available height = terminal height minus:
	//   1 title + 1 blank + 3 topRow + 1 separator + 1 helpBar = 7
	availableHeight := m.height - 7
	if availableHeight < 10 {
		availableHeight = 10
	}

	// Calculate column widths
	leftWidth := (m.width - 3) / 2
	rightWidth := m.width - leftWidth - 3

	leftCol := m.renderMatchupAndAtBat(game)
	rightCol := m.renderPlays(game, availableHeight, rightWidth, false, true)

	leftStyled := lipgloss.NewStyle().
		Width(leftWidth).
		Height(availableHeight).
		Render(leftCol)

	rightStyled := lipgloss.NewStyle().
		Width(rightWidth).
		Height(availableHeight).
		Render(rightCol)

	// Build full-height vertical divider
	dividerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#CCCCCC", Dark: "#444444"})
	dividerLines := make([]string, availableHeight)
	for i := range dividerLines {
		dividerLines[i] = " │ "
	}
	divider := dividerStyle.Render(strings.Join(dividerLines, "\n"))

	bottomSection := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftStyled,
		divider,
		rightStyled,
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		topRow,
		separator,
		bottomSection,
	)
}

// renderCompactGameSituation renders the scoreboard as a 3-line grid:
//
//	▲   B: ●○○○      ◆       1  2  3  4  5  6  7  8  9   R  H  E
//	3   S: ●●○      ◆   ◇   HOU 0  0  1                   1  2  0
//	    O: ●●○               BAL 0  0                       0  1  0
func (m Model) renderCompactGameSituation(game *api.Game) string {
	ls := game.LiveData.Linescore

	liveInningStyle := lipgloss.NewStyle().
		Foreground(colorLiveInning).
		Bold(true)

	// Inning column (3 lines)
	// Arrow above number for top half, below number for bottom half
	inningNum := fmt.Sprintf("%d", ls.CurrentInning)
	var inningLine1, inningLine2, inningLine3 string
	if ls.InningState == "Bottom" {
		inningLine1 = " "
		inningLine2 = liveInningStyle.Render(inningNum)
		inningLine3 = liveInningStyle.Render("▼")
	} else {
		inningLine1 = liveInningStyle.Render("▲")
		inningLine2 = liveInningStyle.Render(inningNum)
		inningLine3 = " "
	}

	// Count column (3 lines: B, S, O) - all 10 visible chars wide
	// B: ○ ○ ○ ○  /  S: ○ ○ ○__  /  O: ○ ○ ○__
	ballsStr := "B:"
	for i := 0; i < 4; i++ {
		ballsStr += " "
		if i < ls.Balls {
			ballsStr += countFilledStyle.Render("●")
		} else {
			ballsStr += ballEmptyStyle.Render("○")
		}
	}
	strikesStr := "S:"
	for i := 0; i < 3; i++ {
		strikesStr += " "
		if i < ls.Strikes {
			strikesStr += strikeFilledStyle.Render("●")
		} else {
			strikesStr += strikeEmptyStyle.Render("○")
		}
	}
	strikesStr += "  "
	outsStr := "O:"
	for i := 0; i < 3; i++ {
		outsStr += " "
		if i < ls.Outs {
			outsStr += outFilledStyle.Render("●")
		} else {
			outsStr += outEmptyStyle.Render("○")
		}
	}
	outsStr += "  "

	// Bases column (3 lines: second on top, third+first on middle, blank)
	second := baseEmptyStyle.Render("◇")
	if ls.Offense.Second != nil {
		second = baseFilledStyle.Render("◆")
	}
	third := baseEmptyStyle.Render("◇")
	if ls.Offense.Third != nil {
		third = baseFilledStyle.Render("◆")
	}
	first := baseEmptyStyle.Render("◇")
	if ls.Offense.First != nil {
		first = baseFilledStyle.Render("◆")
	}
	basesLine1 := "  " + second + "  "
	basesLine2 := third + "   " + first
	basesLine3 := "     "

	// Linescore column (3 lines: header, away, home)
	awayName := game.Teams.Away.Team.Name
	homeName := game.Teams.Home.Team.Name
	if awayName == "" && game.GameData != nil {
		awayName = game.GameData.Teams.Away.Name
	}
	if homeName == "" && game.GameData != nil {
		homeName = game.GameData.Teams.Home.Name
	}
	awayAbbr := getTeamAbbreviation(awayName)
	homeAbbr := getTeamAbbreviation(homeName)
	awayColors := GetTeamColors(awayName)
	homeColors := GetTeamColors(homeName)

	totalInnings := ls.CurrentInning
	if totalInnings < 9 {
		totalInnings = 9
	}

	scoreStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#AAAAAA"})
	rheBold := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#F8F8F2"}).
		Bold(true)

	// Header row
	lsHeader := fmt.Sprintf("%-3s", "")
	for i := 1; i <= totalInnings; i++ {
		lsHeader += fmt.Sprintf(" %2d", i)
	}
	lsHeader += "   "
	lsHeaderStyled := scoreStyle.Render(lsHeader) + rheBold.Render("R  H  E")

	// Away row
	awayAbbrStyle := lipgloss.NewStyle().Foreground(awayColors.Primary).Bold(true)
	awayAbbrStyled := awayAbbrStyle.Render(fmt.Sprintf("%-3s", awayAbbr))

	var awayLineStyled string
	if m.scoreAnim != nil && m.scoreAnim.IsActive() && m.scoreAnim.AwayChanged() {
		awayLineStyled = awayAbbrStyled + m.scoreAnim.AwayView()
	} else {
		awayInnings := ""
		for i := 0; i < totalInnings; i++ {
			if i < len(ls.Innings) {
				awayInnings += fmt.Sprintf(" %2d", ls.Innings[i].Away.RunsVal())
			} else {
				awayInnings += "   "
			}
		}
		awayR := fmt.Sprintf("%2d", ls.Teams.Away.Runs)
		awayHE := fmt.Sprintf(" %2d %2d", ls.Teams.Away.Hits, ls.Teams.Away.Errors)
		awayLineStyled = awayAbbrStyled + scoreStyle.Render(awayInnings+"  ") + awayR + scoreStyle.Render(awayHE)
	}

	// Home row
	homeAbbrStyle := lipgloss.NewStyle().Foreground(homeColors.Primary).Bold(true)
	homeAbbrStyled := homeAbbrStyle.Render(fmt.Sprintf("%-3s", homeAbbr))

	var homeLineStyled string
	if m.scoreAnim != nil && m.scoreAnim.IsActive() && m.scoreAnim.HomeChanged() {
		homeLineStyled = homeAbbrStyled + m.scoreAnim.HomeView()
	} else {
		homeInnings := ""
		for i := 0; i < totalInnings; i++ {
			if i < len(ls.Innings) {
				homeInnings += fmt.Sprintf(" %2d", ls.Innings[i].Home.RunsVal())
			} else {
				homeInnings += "   "
			}
		}
		homeR := fmt.Sprintf("%2d", ls.Teams.Home.Runs)
		homeHE := fmt.Sprintf(" %2d %2d", ls.Teams.Home.Hits, ls.Teams.Home.Errors)
		homeLineStyled = homeAbbrStyled + scoreStyle.Render(homeInnings+"  ") + homeR + scoreStyle.Render(homeHE)
	}

	// Build the 3 lines, padding the left portion so the linescore
	// aligns above the right (all plays) column.
	countBasesGap := "               " // 15 spaces between count and bases
	leftWidth := (m.width-3)/2 + 3       // match left column + divider from renderLiveGameStatus
	padStyle := lipgloss.NewStyle().Width(leftWidth)

	line1Left := inningLine1 + "  " + ballsStr + countBasesGap + basesLine1
	line2Left := inningLine2 + "  " + strikesStr + countBasesGap + basesLine2
	line3Left := inningLine3 + "  " + outsStr + countBasesGap + basesLine3

	line1 := padStyle.Render(line1Left) + lsHeaderStyled
	line2 := padStyle.Render(line2Left) + awayLineStyled
	line3 := padStyle.Render(line3Left) + homeLineStyled

	return line1 + "\n" + line2 + "\n" + line3
}

// renderMatchupAndAtBat renders the matchup and current at-bat (left column)
func (m Model) renderMatchupAndAtBat(game *api.Game) string {
	if game.LiveData.Plays.CurrentPlay == nil {
		return ""
	}

	play := game.LiveData.Plays.CurrentPlay
	var b strings.Builder

	// Get pitcher and batter from boxscore
	pitcherID := play.Matchup.Pitcher.ID
	batterID := play.Matchup.Batter.ID

	pitcherKey := fmt.Sprintf("ID%d", pitcherID)
	batterKey := fmt.Sprintf("ID%d", batterID)

	var pitcher *api.BoxscorePlayer
	var pitcherTeam string
	var batter *api.BoxscorePlayer
	var batterTeam string

	// Find pitcher
	if p, ok := game.LiveData.Boxscore.Teams.Home.Players[pitcherKey]; ok {
		pitcher = &p
		pitcherTeam = getTeamAbbreviation(game.Teams.Home.Team.Name)
	} else if p, ok := game.LiveData.Boxscore.Teams.Away.Players[pitcherKey]; ok {
		pitcher = &p
		pitcherTeam = getTeamAbbreviation(game.Teams.Away.Team.Name)
	}

	// Find batter
	if b, ok := game.LiveData.Boxscore.Teams.Home.Players[batterKey]; ok {
		batter = &b
		batterTeam = getTeamAbbreviation(game.Teams.Home.Team.Name)
	} else if b, ok := game.LiveData.Boxscore.Teams.Away.Players[batterKey]; ok {
		batter = &b
		batterTeam = getTeamAbbreviation(game.Teams.Away.Team.Name)
	}

	// Render matchup
	if pitcher != nil {
		pitchingLine := fmt.Sprintf("%s Pitching: ", pitcherTeam)
		pitchingLine += lipgloss.NewStyle().Bold(true).Render(pitcher.Person.FullName)
		if pitcher.Stats.Pitching != nil {
			pitchingLine += fmt.Sprintf(" %s IP, %d P, %s ERA",
				pitcher.Stats.Pitching.InningsPitched,
				pitcher.Stats.Pitching.PitchesThrown,
				pitcher.SeasonStats.Pitching.Era)
		}
		b.WriteString(pitchingLine)
		b.WriteString("\n")
	}

	if batter != nil {
		battingLine := fmt.Sprintf("%s At Bat:   ", batterTeam)
		battingLine += lipgloss.NewStyle().Bold(true).Render(batter.Person.FullName)
		if batter.Stats.Batting != nil && batter.SeasonStats != nil && batter.SeasonStats.Batting != nil {
			battingLine += fmt.Sprintf(" %d-%d, %s AVG, %d HR",
				batter.Stats.Batting.Hits,
				batter.Stats.Batting.AtBats,
				batter.SeasonStats.Batting.Avg,
				batter.SeasonStats.Batting.HomeRuns)
		}
		b.WriteString(battingLine)
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Current at-bat events
	if len(play.PlayEvents) > 0 {
		// Show play result if complete
		if play.About.IsComplete && play.Result.Description != "" {
			b.WriteString(play.Result.Description)
			b.WriteString("\n\n")
		}

		// Show pitch-by-pitch events (most recent 3, reversed)
		pitchesShown := 0
		maxPitches := 3
		for i := len(play.PlayEvents) - 1; i >= 0 && pitchesShown < maxPitches; i-- {
			event := play.PlayEvents[i]
			if event.IsPitch {
				pitchesShown++
				line := fmt.Sprintf("[%s] ", event.Details.Description)
				if event.PitchData != nil && event.PitchData.StartSpeed > 0 {
					line += fmt.Sprintf("%.1f MPH ", event.PitchData.StartSpeed)
				}
				if event.Details.PitchType != nil && event.Details.PitchType.Description != "" {
					line += event.Details.PitchType.Description
				}
				if !event.Details.IsInPlay {
					line += fmt.Sprintf(" %d-%d", event.Count.Balls, event.Count.Strikes)
				}
				b.WriteString(line)
				b.WriteString("\n")
			} else {
				if event.Details.Event != "" {
					b.WriteString(fmt.Sprintf("[%s] ", event.Details.Event))
				}
				b.WriteString(event.Details.Description)
				b.WriteString("\n")
			}
		}
	}

	// Render strike zone below pitch list
	if len(play.PlayEvents) > 0 {
		zone := m.renderStrikeZone(play.PlayEvents)
		if zone != "" {
			b.WriteString("\n")
			b.WriteString(zone)
		}
	}

	return b.String()
}

// Strike zone grid constants
const (
	szWidth  = 23
	szHeight = 15

	szColLeft  = 2
	szColDiv1  = 8
	szColDiv2  = 14
	szColRight = 20

	szRowTop    = 1
	szRowDiv1   = 5
	szRowDiv2   = 9
	szRowBottom = 13
)

// cellType tracks what occupies each grid cell for styling
type cellType int

const (
	cellEmpty cellType = iota
	cellBorder
	cellBallLatest
	cellBallPrevious
	cellStrikeLatest
	cellStrikePrevious
	cellInPlayLatest
	cellInPlayPrevious
)

func classifyPitch(event api.PlayEvent) cellType {
	if event.Details.IsInPlay {
		return cellInPlayLatest
	}
	if event.Details.IsBall {
		return cellBallLatest
	}
	return cellStrikeLatest
}

func drawZoneGrid(chars [][]rune, types [][]cellType) {
	hCols := []int{szColLeft, szColDiv1, szColDiv2, szColRight}
	hRows := []int{szRowTop, szRowDiv1, szRowDiv2, szRowBottom}

	// Draw horizontal lines
	for _, r := range hRows {
		for c := szColLeft; c <= szColRight; c++ {
			chars[r][c] = '─'
			types[r][c] = cellBorder
		}
	}

	// Draw vertical lines
	for _, c := range hCols {
		for r := szRowTop; r <= szRowBottom; r++ {
			chars[r][c] = '│'
			types[r][c] = cellBorder
		}
	}

	// Draw intersections
	for _, r := range hRows {
		for _, c := range hCols {
			chars[r][c] = '┼'
			types[r][c] = cellBorder
		}
	}

	// Fix corners
	chars[szRowTop][szColLeft] = '┌'
	chars[szRowTop][szColRight] = '┐'
	chars[szRowBottom][szColLeft] = '└'
	chars[szRowBottom][szColRight] = '┘'

	// Fix T-junctions on top edge
	chars[szRowTop][szColDiv1] = '┬'
	chars[szRowTop][szColDiv2] = '┬'

	// Fix T-junctions on bottom edge
	chars[szRowBottom][szColDiv1] = '┴'
	chars[szRowBottom][szColDiv2] = '┴'

	// Fix T-junctions on left edge
	chars[szRowDiv1][szColLeft] = '├'
	chars[szRowDiv2][szColLeft] = '├'

	// Fix T-junctions on right edge
	chars[szRowDiv1][szColRight] = '┤'
	chars[szRowDiv2][szColRight] = '┤'
}

func findZoneBounds(events []api.PlayEvent) (top, bottom float64) {
	top = 3.5
	bottom = 1.5
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].PitchData != nil && events[i].PitchData.StrikeZoneTop > 0 {
			top = events[i].PitchData.StrikeZoneTop
			bottom = events[i].PitchData.StrikeZoneBottom
			return
		}
	}
	return
}

func mapCoordinates(pX, pZ, zoneTop, zoneBottom float64) (col, row int) {
	// Horizontal: plate is 17 inches = 1.417 ft wide, centered at 0
	plateHalf := 0.708
	renderMargin := plateHalf
	renderMin := -plateHalf - renderMargin
	renderMax := plateHalf + renderMargin
	renderRange := renderMax - renderMin

	colF := ((pX - renderMin) / renderRange) * float64(szWidth-1)
	col = int(math.Round(colF))
	col = clampInt(col, 0, szWidth-1)

	// Vertical: higher pZ = lower row number
	zoneH := zoneTop - zoneBottom
	vertMargin := zoneH * (1.0 / 12.0)
	renderTop := zoneTop + vertMargin
	renderBottom := zoneBottom - vertMargin
	vertRange := renderTop - renderBottom

	rowF := ((renderTop - pZ) / vertRange) * float64(szHeight-1)
	row = int(math.Round(rowF))
	row = clampInt(row, 0, szHeight-1)

	return
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func isLatestPitch(events []api.PlayEvent, idx int) bool {
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].IsPitch && events[i].PitchData != nil && events[i].PitchData.Coordinates != nil {
			return i == idx
		}
	}
	return false
}

func styleForCellType(ct cellType) lipgloss.Style {
	switch ct {
	case cellBorder:
		return zoneBorderStyle
	case cellBallLatest:
		return zoneBallStyle
	case cellBallPrevious:
		return zoneBallDimStyle
	case cellStrikeLatest:
		return zoneStrikeStyle
	case cellStrikePrevious:
		return zoneStrikeDimStyle
	case cellInPlayLatest:
		return zoneInPlayStyle
	case cellInPlayPrevious:
		return zoneInPlayDimStyle
	default:
		return zoneEmptyStyle
	}
}

func renderGrid(chars [][]rune, types [][]cellType) string {
	var b strings.Builder
	for r := 0; r < len(chars); r++ {
		if r > 0 {
			b.WriteRune('\n')
		}
		c := 0
		for c < len(chars[r]) {
			ct := types[r][c]
			start := c
			for c < len(chars[r]) && types[r][c] == ct {
				c++
			}
			segment := string(chars[r][start:c])
			b.WriteString(styleForCellType(ct).Render(segment))
		}
	}
	return b.String()
}

func renderPitchLabel(events []api.PlayEvent) string {
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].IsPitch {
			event := events[i]
			var lines []string

			// Pitch type and speed
			var pitchParts []string
			if event.Details.PitchType != nil && event.Details.PitchType.Code != "" {
				pitchParts = append(pitchParts, event.Details.PitchType.Code)
			}
			if event.PitchData != nil && event.PitchData.StartSpeed > 0 {
				pitchParts = append(pitchParts, fmt.Sprintf("%.1f mph", event.PitchData.StartSpeed))
			}
			if len(pitchParts) > 0 {
				lines = append(lines, strings.Join(pitchParts, " "))
			}

			// Hit metrics for balls in play
			if event.Details.IsInPlay && event.HitData != nil {
				if event.HitData.LaunchSpeed > 0 {
					lines = append(lines, fmt.Sprintf("EV %.1f", event.HitData.LaunchSpeed))
				}
				if event.HitData.LaunchAngle != 0 {
					lines = append(lines, fmt.Sprintf("LA %.0f\u00b0", event.HitData.LaunchAngle))
				}
				if event.HitData.TotalDistance > 0 {
					lines = append(lines, fmt.Sprintf("%.0f ft", event.HitData.TotalDistance))
				}
			}

			if len(lines) > 0 {
				return zoneLabelStyle.
					AlignHorizontal(lipgloss.Left).
					Render(strings.Join(lines, "\n"))
			}
			return ""
		}
	}
	return ""
}

func (m Model) renderStrikeZone(events []api.PlayEvent) string {
	chars := make([][]rune, szHeight)
	types := make([][]cellType, szHeight)
	for r := range chars {
		chars[r] = make([]rune, szWidth)
		types[r] = make([]cellType, szWidth)
		for c := range chars[r] {
			chars[r][c] = ' '
			types[r][c] = cellEmpty
		}
	}

	drawZoneGrid(chars, types)

	zoneTop, zoneBottom := findZoneBounds(events)

	pitchCount := 0
	for i, event := range events {
		if !event.IsPitch || event.PitchData == nil || event.PitchData.Coordinates == nil {
			continue
		}
		pitchCount++
		latest := isLatestPitch(events, i)
		col, row := mapCoordinates(
			event.PitchData.Coordinates.PX,
			event.PitchData.Coordinates.PZ,
			zoneTop, zoneBottom,
		)
		marker := '·'
		ct := classifyPitch(event)
		if latest {
			marker = '●'
		} else {
			// Downgrade to previous (dim) variant
			ct++
		}
		if row >= 0 && row < szHeight && col >= 0 && col < szWidth {
			chars[row][col] = marker
			types[row][col] = ct
		}
	}

	if pitchCount == 0 {
		return ""
	}

	grid := renderGrid(chars, types)
	label := renderPitchLabel(events)
	if label != "" {
		spacer := lipgloss.NewStyle().PaddingLeft(1).Render(label)
		return lipgloss.JoinHorizontal(lipgloss.Top, grid, spacer)
	}
	return grid
}

// renderFinalGame renders a finished game
func (m Model) renderFinalGame(game *api.Game) string {
	var sections []string

	// Fixed header (always visible)
	sections = append(sections, m.renderScoreboard(game))

	if game.LiveData != nil {
		sections = append(sections, m.renderLineScore(game))

		decisions := m.renderDecisions(game)
		if decisions != "" {
			sections = append(sections, decisions)
		}
	}

	header := strings.Join(sections, "\n")
	headerHeight := strings.Count(header, "\n") + 1
	availableHeight := m.height - headerHeight - 1 // 1 for help bar

	switch m.gameSubview {
	case AllPlaysSubview:
		return header + "\n" + m.renderPlays(game, availableHeight, m.width, false, false)
	case ScoringPlaysSubview:
		return header + "\n" + m.renderPlays(game, availableHeight, m.width, true, false)
	default:
		return header + "\n" + m.renderBoxScore(game)
	}
}

// renderTeamRecords renders team records
func (m Model) renderTeamRecords(game *api.Game) string {
	awayRecord := fmt.Sprintf("%s (%d-%d)",
		GetTeamShortName(game.Teams.Away.Team.Name),
		game.Teams.Away.LeagueRecord.Wins,
		game.Teams.Away.LeagueRecord.Losses,
	)
	homeRecord := fmt.Sprintf("%s (%d-%d)",
		GetTeamShortName(game.Teams.Home.Team.Name),
		game.Teams.Home.LeagueRecord.Wins,
		game.Teams.Home.LeagueRecord.Losses,
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		itemStyle.Render(awayRecord),
		itemStyle.Render(homeRecord),
	)
}

// renderGameSituation renders inning, count, bases, score
func (m Model) renderGameSituation(game *api.Game) string {
	ls := game.LiveData.Linescore

	// Inning
	inningState := "Mid"
	arrow := ""
	if ls.InningState == "Top" {
		inningState = "Top"
		arrow = "▲"
	} else if ls.InningState == "Bottom" {
		inningState = "Bot"
		arrow = "▼"
	}
	inning := fmt.Sprintf("%s %s %s", inningState, ls.CurrentInningOrdinal, arrow)

	// Count (balls-strikes, outs)
	count := m.renderCount(ls.Balls, ls.Strikes, ls.Outs)

	// Bases
	bases := m.renderBases(ls.Offense)

	// Score
	score := fmt.Sprintf("%s: %d  %s: %d",
		GetTeamShortName(game.Teams.Away.Team.Name), ls.Teams.Away.Runs,
		GetTeamShortName(game.Teams.Home.Team.Name), ls.Teams.Home.Runs,
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		inningStyle.Render(inning),
		count,
		bases,
		scoreStyle.Render(score),
	)
}

// renderCount renders balls, strikes, outs
func (m Model) renderCount(balls, strikes, outs int) string {
	ballsStr := "Balls: "
	for i := 0; i < 4; i++ {
		if i < balls {
			ballsStr += countFilledStyle.Render("●")
		} else {
			ballsStr += ballEmptyStyle.Render("○")
		}
	}

	strikesStr := "  Strikes: "
	for i := 0; i < 3; i++ {
		if i < strikes {
			strikesStr += strikeFilledStyle.Render("●")
		} else {
			strikesStr += strikeEmptyStyle.Render("○")
		}
	}

	outsStr := "  Outs: "
	for i := 0; i < 3; i++ {
		if i < outs {
			outsStr += outFilledStyle.Render("●")
		} else {
			outsStr += outEmptyStyle.Render("○")
		}
	}

	return itemStyle.Render(ballsStr + strikesStr + outsStr)
}

// renderBases renders base runner status
func (m Model) renderBases(offense api.Offense) string {
	first := "○"
	second := "○"
	third := "○"

	if offense.First != nil {
		first = baseFilledStyle.Render("◆")
	} else {
		first = baseEmptyStyle.Render("◇")
	}
	if offense.Second != nil {
		second = baseFilledStyle.Render("◆")
	} else {
		second = baseEmptyStyle.Render("◇")
	}
	if offense.Third != nil {
		third = baseFilledStyle.Render("◆")
	} else {
		third = baseEmptyStyle.Render("◇")
	}

	// Diamond layout
	bases := fmt.Sprintf("  %s\n %s %s\n  %s", second, third, first, "◇")
	return itemStyle.Render(bases)
}

// renderCurrentPlay renders the current at-bat
func (m Model) renderCurrentPlay(game *api.Game) string {
	if game.LiveData.Plays.CurrentPlay == nil {
		return ""
	}

	play := game.LiveData.Plays.CurrentPlay

	var b strings.Builder
	b.WriteString(headerStyle.Render("Current At-Bat"))
	b.WriteString("\n")

	// Matchup
	matchup := fmt.Sprintf("%s vs %s",
		play.Matchup.Batter.FullName,
		play.Matchup.Pitcher.FullName,
	)
	b.WriteString(itemStyle.Render(matchup))
	b.WriteString("\n")

	// Last few pitches
	if len(play.PlayEvents) > 0 {
		start := len(play.PlayEvents) - 5
		if start < 0 {
			start = 0
		}
		for _, event := range play.PlayEvents[start:] {
			if event.IsPitch {
				pitch := fmt.Sprintf("%s - %s (%.1f mph)",
					event.Details.Code,
					event.Details.Description,
					event.Details.StartSpeed,
				)
				b.WriteString(itemStyle.Render(pitch))
				b.WriteString("\n")
			}
		}
	}

	// Result if complete
	if play.About.IsComplete {
		b.WriteString(liveStyle.Render(play.Result.Description))
	}

	return b.String()
}

// renderLineScore renders the inning-by-inning line score in a panel
func (m Model) renderLineScore(game *api.Game) string {
	ls := game.LiveData.Linescore

	var b strings.Builder

	// Get team names with fallback
	awayName := game.Teams.Away.Team.Name
	homeName := game.Teams.Home.Team.Name
	if awayName == "" && game.GameData != nil {
		awayName = game.GameData.Teams.Away.Name
	}
	if homeName == "" && game.GameData != nil {
		homeName = game.GameData.Teams.Home.Name
	}

	// Use abbreviated team names for the line score
	awayAbbr := getTeamAbbreviation(awayName)
	homeAbbr := getTeamAbbreviation(homeName)

	// Get team colors
	awayColors := GetTeamColors(awayName)
	homeColors := GetTeamColors(homeName)

	// Determine total columns: at least 9, or more for extra innings
	totalInnings := len(ls.Innings)
	if totalInnings < 9 {
		totalInnings = 9
	}

	isFinal := game.Status.AbstractGameState == "Final"
	if !isFinal && game.GameData != nil {
		isFinal = game.GameData.Status.AbstractGameState == "Final"
	}

	// Header
	header := fmt.Sprintf("%-3s", "")
	for i := 1; i <= totalInnings; i++ {
		header += fmt.Sprintf("%3d", i)
	}
	header += fmt.Sprintf("   %3s %3s %3s", "R", "H", "E")

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}).
		Bold(true)
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")

	// Away team row - gradient abbreviation + score data
	awayAbbrStyled := anim.BlendGradientBold(fmt.Sprintf("%-3s", awayAbbr), awayColors.Primary, awayColors.Secondary)
	awayScores := ""
	for i := 0; i < totalInnings; i++ {
		if i < len(ls.Innings) && ls.Innings[i].Away.WasPlayed() {
			awayScores += fmt.Sprintf("%3d", ls.Innings[i].Away.RunsVal())
		} else if isFinal {
			awayScores += "  X"
		} else {
			awayScores += "   "
		}
	}
	awayScores += fmt.Sprintf("   %3d %3d %3d",
		ls.Teams.Away.Runs,
		ls.Teams.Away.Hits,
		ls.Teams.Away.Errors,
	)

	scoreStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#333333", Dark: "#CCCCCC"})
	b.WriteString(awayAbbrStyled + scoreStyle.Render(awayScores))
	b.WriteString("\n")

	// Home team row - gradient abbreviation + score data
	homeAbbrStyled := anim.BlendGradientBold(fmt.Sprintf("%-3s", homeAbbr), homeColors.Primary, homeColors.Secondary)
	homeScores := ""
	for i := 0; i < totalInnings; i++ {
		if i < len(ls.Innings) && ls.Innings[i].Home.WasPlayed() {
			homeScores += fmt.Sprintf("%3d", ls.Innings[i].Home.RunsVal())
		} else if isFinal {
			homeScores += "  X"
		} else {
			homeScores += "   "
		}
	}
	homeScores += fmt.Sprintf("   %3d %3d %3d",
		ls.Teams.Home.Runs,
		ls.Teams.Home.Hits,
		ls.Teams.Home.Errors,
	)

	b.WriteString(homeAbbrStyled + scoreStyle.Render(homeScores))

	// Wrap in panel with border
	linescore := b.String()
	panel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.AdaptiveColor{Light: "#CCCCCC", Dark: "#444444"}).
		Padding(0, 1).
		Align(lipgloss.Center).
		Render(linescore)

	// Center on screen
	centered := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		Render(panel)

	return centered
}

// getTeamAbbreviation returns a 2-3 letter abbreviation for a team name
func getTeamAbbreviation(teamName string) string {
	abbrs := map[string]string{
		"Yankees":      "NYY",
		"Red Sox":      "BOS",
		"Blue Jays":    "TOR",
		"Rays":         "TB",
		"Orioles":      "BAL",
		"White Sox":    "CHW",
		"Guardians":    "CLE",
		"Tigers":       "DET",
		"Royals":       "KC",
		"Twins":        "MIN",
		"Astros":       "HOU",
		"Angels":       "LAA",
		"Athletics":    "OAK",
		"Mariners":     "SEA",
		"Rangers":      "TEX",
		"Braves":       "ATL",
		"Marlins":      "MIA",
		"Mets":         "NYM",
		"Phillies":     "PHI",
		"Nationals":    "WSH",
		"Cubs":         "CHC",
		"Reds":         "CIN",
		"Brewers":      "MIL",
		"Pirates":      "PIT",
		"Cardinals":    "STL",
		"Diamondbacks": "ARI",
		"Rockies":      "COL",
		"Dodgers":      "LAD",
		"Padres":       "SD",
		"Giants":       "SF",

		// WBC national teams
		"Australia":                  "AUS",
		"Brazil":                     "BRA",
		"Canada":                     "CAN",
		"Chinese Taipei":             "TPE",
		"Colombia":                   "COL",
		"Cuba":                       "CUB",
		"Czechia":                    "CZE",
		"Dominican Republic":         "DOM",
		"Great Britain":              "GBR",
		"Israel":                     "ISR",
		"Italy":                      "ITA",
		"Japan":                      "JPN",
		"Korea":                      "KOR",
		"Kingdom of the Netherlands": "NED",
		"Mexico":                     "MEX",
		"Nicaragua":                  "NCA",
		"Panama":                     "PAN",
		"Puerto Rico":                "PUR",
		"United States":              "USA",
		"Venezuela":                  "VEN",
	}

	// Try to find abbreviation
	for key, abbr := range abbrs {
		if strings.Contains(teamName, key) {
			return abbr
		}
	}

	// Fallback: take first 3 letters
	if len(teamName) >= 3 {
		return strings.ToUpper(teamName[:3])
	}
	return strings.ToUpper(teamName)
}

// renderPlays renders scrollable play history, truncated to fit availableHeight.
// scoringOnly filters to only scoring plays. reverse shows newest first (true) or oldest first (false).
func (m Model) renderPlays(game *api.Game, availableHeight int, colWidth int, scoringOnly bool, reverse bool) string {
	var b strings.Builder

	plays := game.LiveData.Plays.AllPlays
	if len(plays) == 0 {
		if scoringOnly {
			b.WriteString(itemStyle.Render("No scoring plays yet"))
		} else {
			b.WriteString(itemStyle.Render("No plays yet"))
		}
		return b.String()
	}

	// Build iteration order
	start, end, step := 0, len(plays), 1
	if reverse {
		start, end, step = len(plays)-1, -1, -1
	}

	// Build all play lines into a flat list
	var allLines []string
	currentInning := 0
	currentHalf := ""
	hasPlays := false

	for i := start; i != end; i += step {
		play := plays[i]

		if scoringOnly && !play.About.IsScoringPlay {
			continue
		}
		hasPlays = true

		// Inning header when inning changes
		halfInning := strings.ToUpper(play.About.HalfInning)
		if play.About.Inning != currentInning || play.About.HalfInning != currentHalf {
			if currentInning != 0 {
				allLines = append(allLines, "")
			}
			currentInning = play.About.Inning
			currentHalf = play.About.HalfInning
			inningHeader := fmt.Sprintf("[%s %d]", halfInning, currentInning)
			inningHeaderStyle := lipgloss.NewStyle().
				Foreground(colorInningHeader).
				Bold(true)
			allLines = append(allLines, inningHeaderStyle.Render(inningHeader))
		}

		// Event name in brackets, color-coded by result
		eventType := play.Result.Event
		if eventType == "" {
			eventType = play.Result.EventType
		}

		playLine := ""
		if eventType != "" {
			var eventColor lipgloss.TerminalColor = colorDefaultEvent
			if len(play.PlayEvents) > 0 {
				lastEvent := play.PlayEvents[len(play.PlayEvents)-1].Details
				if lastEvent.IsBall {
					eventColor = colorWalk
				} else if lastEvent.IsStrike {
					eventColor = colorStrikeout
				} else if lastEvent.IsInPlay && !play.About.HasOut {
					eventColor = colorInPlayNoOut
				} else if lastEvent.IsInPlay {
					eventColor = colorInPlayOut
				}
			}
			eventStyle := lipgloss.NewStyle().Foreground(eventColor)
			playLine = eventStyle.Render(fmt.Sprintf("[%s]", eventType)) + " "
		}

		// Description with scoring plays highlighted
		if play.About.IsScoringPlay {
			playLine += lipgloss.NewStyle().
				Foreground(colorScoringPlay).
				Render(play.Result.Description)
			scoreText := fmt.Sprintf("%d - %d",
				play.Result.AwayScore,
				play.Result.HomeScore)
			scoreStyle := lipgloss.NewStyle().
				Bold(true).
				Padding(0, 1).
				Foreground(colorScoreBadgeFg).
				Background(colorScoreBadgeBg)
			playLine += " " + scoreStyle.Render(scoreText)
		} else {
			playLine += play.Result.Description
		}

		// Out tracking
		if play.About.HasOut && play.About.IsComplete {
			currentOuts := play.Count.Outs
			prevOuts := 0
			if i > 0 {
				prevPlay := plays[i-1]
				if prevPlay.About.Inning == play.About.Inning && prevPlay.About.HalfInning == play.About.HalfInning {
					prevOuts = prevPlay.Count.Outs
				}
			}
			if currentOuts > prevOuts {
				outText := fmt.Sprintf(" %d out", currentOuts)
				if currentOuts > 1 {
					outText += "s"
				}
				outStyle := lipgloss.NewStyle().Bold(true)
				playLine += outStyle.Render(outText)
			}
		}

		allLines = append(allLines, playLine)

		// Action events (substitutions, challenges, etc.)
		if !scoringOnly {
			actionStyle := lipgloss.NewStyle().Foreground(colorActionEvent)
			if reverse {
				for j := len(play.PlayEvents) - 1; j >= 0; j-- {
					ev := play.PlayEvents[j]
					if ev.Type != "action" {
						continue
					}
					actionLine := ""
					if ev.Details.Event != "" {
						actionLine = actionStyle.Render(fmt.Sprintf("[%s]", ev.Details.Event)) + " "
					}
					actionLine += ev.Details.Description
					allLines = append(allLines, actionLine)
				}
			} else {
				for _, ev := range play.PlayEvents {
					if ev.Type != "action" {
						continue
					}
					actionLine := ""
					if ev.Details.Event != "" {
						actionLine = actionStyle.Render(fmt.Sprintf("[%s]", ev.Details.Event)) + " "
					}
					actionLine += ev.Details.Description
					allLines = append(allLines, actionLine)
				}
			}
		}
	}

	if scoringOnly && !hasPlays {
		b.WriteString(itemStyle.Render("No scoring plays yet"))
		return b.String()
	}

	// Pre-wrap all logical lines to colWidth and flatten into terminal lines.
	// This makes viewport height deterministic (no estimation drift).
	wrapStyle := lipgloss.NewStyle().Width(colWidth)
	var termLines []string
	for _, line := range allLines {
		if line == "" {
			termLines = append(termLines, "")
			continue
		}
		wrapped := wrapStyle.Render(line)
		for _, tl := range strings.Split(wrapped, "\n") {
			termLines = append(termLines, tl)
		}
	}

	// Reserve lines for scroll indicators (up and/or down)
	reservedLines := 0
	if m.gameScrollOffset > 0 {
		reservedLines++
	}
	if len(termLines) > availableHeight {
		reservedLines++
	}
	viewHeight := availableHeight - reservedLines
	if viewHeight < 1 {
		viewHeight = 1
	}

	// Clamp scroll offset
	maxOffset := len(termLines) - viewHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.gameScrollOffset > maxOffset {
		m.gameScrollOffset = maxOffset
	}
	if m.gameScrollOffset < 0 {
		m.gameScrollOffset = 0
	}

	// Slice the exact viewport
	viewEnd := m.gameScrollOffset + viewHeight
	if viewEnd > len(termLines) {
		viewEnd = len(termLines)
	}

	// Get team colors for indicators
	away, home := getGameTeamColors(game)

	if m.gameScrollOffset > 0 {
		b.WriteString(anim.ScrollIndicator(anim.ScrollUp, m.gameScrollOffset, colWidth, away, home))
		b.WriteString("\n")
	}

	for _, tl := range termLines[m.gameScrollOffset:viewEnd] {
		b.WriteString(tl)
		b.WriteString("\n")
	}

	if viewEnd < len(termLines) {
		remaining := len(termLines) - viewEnd
		b.WriteString(anim.ScrollIndicator(anim.ScrollDown, remaining, colWidth, away, home))
	}

	return b.String()
}

// renderScoreboard renders a sophisticated scoreboard panel with team colors
func (m Model) renderScoreboard(game *api.Game) string {
	// Get scores from linescore if available
	awayRuns := game.Teams.Away.Score
	homeRuns := game.Teams.Home.Score
	if game.LiveData != nil {
		awayRuns = game.LiveData.Linescore.Teams.Away.Runs
		homeRuns = game.LiveData.Linescore.Teams.Home.Runs
	}

	// Get team names and records
	awayFull := game.Teams.Away.Team.Name
	homeFull := game.Teams.Home.Team.Name
	if awayFull == "" && game.GameData != nil {
		awayFull = game.GameData.Teams.Away.Name
	}
	if homeFull == "" && game.GameData != nil {
		homeFull = game.GameData.Teams.Home.Name
	}
	awayName := GetTeamShortName(awayFull)
	homeName := GetTeamShortName(homeFull)

	// Get team colors
	awayColors := GetTeamColors(awayFull)
	homeColors := GetTeamColors(homeFull)

	awayRecord := ""
	homeRecord := ""

	if game.GameData != nil {
		// Use series record for playoffs, otherwise use league record
		if game.GameData.Teams.Away.Record != nil {
			awayRecord = fmt.Sprintf("%d-%d",
				game.GameData.Teams.Away.Record.Wins,
				game.GameData.Teams.Away.Record.Losses)
		} else {
			awayRecord = fmt.Sprintf("%d-%d",
				game.GameData.Teams.Away.LeagueRecord.Wins,
				game.GameData.Teams.Away.LeagueRecord.Losses)
		}

		if game.GameData.Teams.Home.Record != nil {
			homeRecord = fmt.Sprintf("%d-%d",
				game.GameData.Teams.Home.Record.Wins,
				game.GameData.Teams.Home.Record.Losses)
		} else {
			homeRecord = fmt.Sprintf("%d-%d",
				game.GameData.Teams.Home.LeagueRecord.Wins,
				game.GameData.Teams.Home.LeagueRecord.Losses)
		}
	}

	// Create team panels
	awayPanel := lipgloss.NewStyle().
		Width(30).
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(awayColors.Primary).
		Align(lipgloss.Center).
		Render(lipgloss.JoinVertical(lipgloss.Center,
			lipgloss.NewStyle().Foreground(awayColors.Primary).Bold(true).Render(awayName),
			lipgloss.NewStyle().Foreground(awayColors.Secondary).Render(awayRecord),
		))

	homePanel := lipgloss.NewStyle().
		Width(30).
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(homeColors.Primary).
		Align(lipgloss.Center).
		Render(lipgloss.JoinVertical(lipgloss.Center,
			lipgloss.NewStyle().Foreground(homeColors.Primary).Bold(true).Render(homeName),
			lipgloss.NewStyle().Foreground(homeColors.Secondary).Render(homeRecord),
		))

	// Create score panel in the center
	scorePanel := lipgloss.NewStyle().
		Width(20).
		Padding(0, 2).
		Border(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#AAAAAA"}).
		Align(lipgloss.Center).
		Render(lipgloss.JoinVertical(lipgloss.Center,
			lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}).Render("FINAL"),
			lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#F8F8F2"}).
				Bold(true).
				Render(fmt.Sprintf("%d - %d", awayRuns, homeRuns)),
		))

	// Join panels horizontally
	scoreboard := lipgloss.JoinHorizontal(
		lipgloss.Center,
		awayPanel,
		scorePanel,
		homePanel,
	)

	centered := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		Render(scoreboard)

	return centered
}

// renderFinalScore renders the final score (legacy, keeping for compatibility)
func (m Model) renderFinalScore(game *api.Game) string {
	// Get scores from linescore if available, fallback to team score
	awayRuns := game.Teams.Away.Score
	homeRuns := game.Teams.Home.Score
	if game.LiveData != nil {
		awayRuns = game.LiveData.Linescore.Teams.Away.Runs
		homeRuns = game.LiveData.Linescore.Teams.Home.Runs
	}

	awayScore := fmt.Sprintf("%s: %d",
		GetTeamShortName(game.Teams.Away.Team.Name),
		awayRuns,
	)
	homeScore := fmt.Sprintf("%s: %d",
		GetTeamShortName(game.Teams.Home.Team.Name),
		homeRuns,
	)

	// Determine winner from scores
	if awayRuns > homeRuns {
		awayScore = scoreStyle.Render(awayScore + " (W)")
	} else if homeRuns > awayRuns {
		homeScore = scoreStyle.Render(homeScore + " (W)")
	}

	return lipgloss.JoinVertical(lipgloss.Left, awayScore, homeScore)
}

// renderDecisions renders win/loss/save decisions
func (m Model) renderDecisions(game *api.Game) string {
	if game.LiveData == nil || game.LiveData.Decisions.Winner == nil {
		return ""
	}

	var parts []string

	decisions := game.LiveData.Decisions
	boxscore := game.LiveData.Boxscore

	// Winner
	if decisions.Winner != nil {
		pitcher := getPlayerFromBoxscore(decisions.Winner.ID, boxscore)
		if pitcher != nil && pitcher.SeasonStats != nil && pitcher.SeasonStats.Pitching != nil {
			stats := pitcher.SeasonStats.Pitching
			parts = append(parts, fmt.Sprintf("Win:  %s (%d-%d)",
				pitcher.Person.FullName, stats.Wins, stats.Losses))
		} else {
			parts = append(parts, fmt.Sprintf("Win:  %s", decisions.Winner.FullName))
		}
	}

	// Loser
	if decisions.Loser != nil {
		pitcher := getPlayerFromBoxscore(decisions.Loser.ID, boxscore)
		if pitcher != nil && pitcher.SeasonStats != nil && pitcher.SeasonStats.Pitching != nil {
			stats := pitcher.SeasonStats.Pitching
			parts = append(parts, fmt.Sprintf("Loss: %s (%d-%d)",
				pitcher.Person.FullName, stats.Wins, stats.Losses))
		} else {
			parts = append(parts, fmt.Sprintf("Loss: %s", decisions.Loser.FullName))
		}
	}

	// Save
	if decisions.Save != nil {
		pitcher := getPlayerFromBoxscore(decisions.Save.ID, boxscore)
		if pitcher != nil && pitcher.SeasonStats != nil && pitcher.SeasonStats.Pitching != nil {
			stats := pitcher.SeasonStats.Pitching
			parts = append(parts, fmt.Sprintf("Save: %s (%d)",
				pitcher.Person.FullName, stats.Saves))
		} else {
			parts = append(parts, fmt.Sprintf("Save: %s", decisions.Save.FullName))
		}
	}

	// Center the decisions on screen
	decisionsText := strings.Join(parts, "\n")
	centered := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		Render(decisionsText)

	return centered
}

// getPlayerFromBoxscore finds a player in the boxscore by ID
func getPlayerFromBoxscore(playerID int, boxscore api.Boxscore) *api.BoxscorePlayer {
	// Check home team
	for _, player := range boxscore.Teams.Home.Players {
		if player.Person.ID == playerID {
			return &player
		}
	}
	// Check away team
	for _, player := range boxscore.Teams.Away.Players {
		if player.Person.ID == playerID {
			return &player
		}
	}
	return nil
}

// getGameTeamColors extracts away and home team colors from a game.
// buildScoreLineText builds the plain-text score line (no ANSI styling) for
// one team. The result matches the width used by the score animation:
// totalInnings*3 + 10 characters.
func buildScoreLineText(ls api.Linescore, totalInnings int, homeAway string) string {
	var innings string
	for i := 0; i < totalInnings; i++ {
		if i < len(ls.Innings) {
			var score api.InningScore
			if homeAway == "away" {
				score = ls.Innings[i].Away
			} else {
				score = ls.Innings[i].Home
			}
			innings += fmt.Sprintf(" %2d", score.RunsVal())
		} else {
			innings += "   "
		}
	}
	var team api.LinescoreTeam
	if homeAway == "away" {
		team = ls.Teams.Away
	} else {
		team = ls.Teams.Home
	}
	return innings + fmt.Sprintf("  %2d %2d %2d", team.Runs, team.Hits, team.Errors)
}

func getGameTeamColors(game *api.Game) (away, home lipgloss.Color) {
	ac, hc := getGameTeamColorsFull(game)
	return ac.Primary, hc.Primary
}

func getGameTeamColorsFull(game *api.Game) (away, home TeamColors) {
	awayName := game.Teams.Away.Team.Name
	homeName := game.Teams.Home.Team.Name
	if awayName == "" && game.GameData != nil {
		awayName = game.GameData.Teams.Away.Name
	}
	if homeName == "" && game.GameData != nil {
		homeName = game.GameData.Teams.Home.Name
	}
	return GetTeamColors(awayName), GetTeamColors(homeName)
}
