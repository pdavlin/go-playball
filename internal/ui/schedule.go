package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pdavlin/go-playball/internal/api"
	"github.com/pdavlin/go-playball/internal/ui/anim"
)

const minCardWidth = 34

// isGameLive checks whether a game is live, checking both the top-level status
// (populated by the schedule endpoint) and the nested gameData status (populated
// by the live feed endpoint).
func isGameLive(game *api.Game) bool {
	if game.Status.AbstractGameState == "Live" {
		return true
	}
	if game.GameData != nil && game.GameData.Status.AbstractGameState == "Live" {
		return true
	}
	return false
}

// getGameLinescore returns the linescore from either the schedule hydration or live data
func getGameLinescore(game api.Game) *api.Linescore {
	if game.LiveData != nil {
		return &game.LiveData.Linescore
	}
	return game.Linescore
}

// sortGames sorts games: Live > Preview > Final, then by inning for live games
func sortGames(games []api.Game) {
	sort.SliceStable(games, func(i, j int) bool {
		stateOrder := map[string]int{"Live": 0, "Preview": 1, "Final": 2}
		si := stateOrder[games[i].Status.AbstractGameState]
		sj := stateOrder[games[j].Status.AbstractGameState]
		if si != sj {
			return si < sj
		}
		if games[i].Status.AbstractGameState == "Live" {
			li := getGameLinescore(games[i])
			lj := getGameLinescore(games[j])
			if li != nil && lj != nil {
				if li.CurrentInning != lj.CurrentInning {
					return li.CurrentInning > lj.CurrentInning
				}
				if li.InningState != lj.InningState {
					return li.InningState == "Bottom" || li.InningState == "End"
				}
			}
		}
		return false
	})
}

// scheduleGridCols returns the number of columns for the schedule grid
func scheduleGridCols(width int) int {
	numCols := width / minCardWidth
	if numCols < 1 {
		numCols = 1
	}
	return numCols
}

// scheduleVisibleRows returns how many rows fit on screen
func scheduleVisibleRows(height int) int {
	// Each card is 5 lines: 3 content + 2 border
	// Reserve 5 lines: title(1) + newline(1) + helpbar(1) + scroll indicators(2)
	cardHeight := 5
	availableHeight := height - 5
	rows := availableHeight / cardHeight
	if rows < 1 {
		rows = 1
	}
	return rows
}

// handleScheduleKeys handles keyboard input for schedule view
func (m Model) handleScheduleKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	numCols := scheduleGridCols(m.width)
	visibleRows := scheduleVisibleRows(m.height)

	switch msg.String() {
	case "up", "k":
		next := m.selectedGameIdx - numCols
		if next >= 0 {
			m.selectedGameIdx = next
		}
	case "down", "j":
		next := m.selectedGameIdx + numCols
		if next < len(m.games) {
			m.selectedGameIdx = next
		}
	case "left", "h":
		if m.selectedGameIdx > 0 {
			m.selectedGameIdx--
		}
	case "right", "l":
		if m.selectedGameIdx < len(m.games)-1 {
			m.selectedGameIdx++
		}
	case "enter":
		if len(m.games) > 0 && m.selectedGameIdx < len(m.games) {
			m.view = GameView
			m.currentGame = nil
			m.expectedGameID = m.games[m.selectedGameIdx].ID
			m.gameRawJSON = nil
			m.gameTimestamp = ""
			m.loading = true
			m.gameScrollOffset = 0
			m.focusedPanel = 0
			m.panelScrollOffsets = [4]int{}
			if m.games[m.selectedGameIdx].Status.AbstractGameState == "Final" {
				m.gameSubview = BoxScoreSubview
			} else {
				m.gameSubview = GameStatusSubview
			}
			awayColors := GetTeamColors(m.games[m.selectedGameIdx].Teams.Away.Team.Name)
			homeColors := GetTeamColors(m.games[m.selectedGameIdx].Teams.Home.Team.Name)
			spinnerCmd := m.startSpinner("Loading", awayColors.Primary, homeColors.Primary)
			return m, tea.Batch(spinnerCmd, loadGameIncremental(m.apiClient, m.games[m.selectedGameIdx].ID, nil, ""))
		}
	case "p":
		m.scheduleDate = m.scheduleDate.AddDate(0, 0, -1)
		m.selectedGameIdx = 0
		m.scheduleScrollOffset = 0
		m.loading = true
		spinnerCmd := m.startSpinner("Loading", colorPrimary, colorAccent)
		return m, tea.Batch(spinnerCmd, loadSchedule(m.apiClient, m.scheduleDate))
	case "n":
		m.scheduleDate = m.scheduleDate.AddDate(0, 0, 1)
		m.selectedGameIdx = 0
		m.scheduleScrollOffset = 0
		m.loading = true
		spinnerCmd := m.startSpinner("Loading", colorPrimary, colorAccent)
		return m, tea.Batch(spinnerCmd, loadSchedule(m.apiClient, m.scheduleDate))
	case "t":
		m.scheduleDate = time.Now()
		m.selectedGameIdx = 0
		m.scheduleScrollOffset = 0
		m.loading = true
		spinnerCmd := m.startSpinner("Loading", colorPrimary, colorAccent)
		return m, tea.Batch(spinnerCmd, loadSchedule(m.apiClient, m.scheduleDate))
	}

	// Keep selected game in bounds
	if m.selectedGameIdx >= len(m.games) {
		m.selectedGameIdx = len(m.games) - 1
	}
	if m.selectedGameIdx < 0 {
		m.selectedGameIdx = 0
	}

	// Adjust scroll offset to keep selected row visible
	selectedRow := m.selectedGameIdx / numCols
	if selectedRow < m.scheduleScrollOffset {
		m.scheduleScrollOffset = selectedRow
	}
	if selectedRow >= m.scheduleScrollOffset+visibleRows {
		m.scheduleScrollOffset = selectedRow - visibleRows + 1
	}

	return m, nil
}

// renderSchedule renders the schedule view as a responsive grid
func (m Model) renderSchedule() string {
	var b strings.Builder

	dateStr := m.scheduleDate.Format("Monday, January 2, 2006")
	titleText := fmt.Sprintf("MLB Schedule - %s", dateStr)

	titlePanel := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#F8F8F2"}).
		Background(lipgloss.AdaptiveColor{Light: "#E8E8E8", Dark: "#1A1A1A"}).
		Padding(0, 2).
		Width(m.width).
		Align(lipgloss.Center).
		Render(titleText)

	b.WriteString(titlePanel)
	b.WriteString("\n")

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		return b.String()
	}

	if len(m.games) == 0 {
		emptyMsg := lipgloss.NewStyle().
			Padding(2).
			Align(lipgloss.Center).
			Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}).
			Render("No games scheduled for this date.")
		b.WriteString(emptyMsg)
		return b.String()
	}

	numCols := scheduleGridCols(m.width)
	// Each card's border adds 2 chars beyond its Width, so budget for that
	cardWidth := (m.width - numCols*2) / numCols
	visibleRows := scheduleVisibleRows(m.height)

	// Total rows
	totalRows := (len(m.games) + numCols - 1) / numCols

	// Clamp scroll offset
	if m.scheduleScrollOffset > totalRows-visibleRows {
		m.scheduleScrollOffset = totalRows - visibleRows
	}
	if m.scheduleScrollOffset < 0 {
		m.scheduleScrollOffset = 0
	}

	startRow := m.scheduleScrollOffset
	endRow := startRow + visibleRows
	if endRow > totalRows {
		endRow = totalRows
	}

	// Render visible rows
	var rows []string
	for row := startRow; row < endRow; row++ {
		var cards []string
		for col := 0; col < numCols; col++ {
			idx := row*numCols + col
			if idx >= len(m.games) {
				// Empty cell to fill the row
				cards = append(cards, lipgloss.NewStyle().Width(cardWidth).Render(""))
				continue
			}
			game := m.games[idx]
			card := m.formatGameCard(game, idx == m.selectedGameIdx, cardWidth)
			cards = append(cards, card)
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cards...))
	}

	gradFrom, gradTo := getScheduleGradientColors(m)
	if m.scheduleScrollOffset > 0 {
		upRemaining := m.scheduleScrollOffset * 5
		b.WriteString(anim.ScrollIndicator(anim.ScrollUp, upRemaining, m.width, gradFrom, gradTo))
		b.WriteString("\n")
	}

	b.WriteString(lipgloss.JoinVertical(lipgloss.Left, rows...))

	if endRow < totalRows {
		downRemaining := (totalRows - endRow) * 5
		b.WriteString("\n")
		b.WriteString(anim.ScrollIndicator(anim.ScrollDown, downRemaining, m.width, gradFrom, gradTo))
	}

	return b.String()
}

// formatGameCard formats a single game as a compact grid card
func (m Model) formatGameCard(game api.Game, selected bool, cardWidth int) string {
	awayFull := game.Teams.Away.Team.Name
	homeFull := game.Teams.Home.Team.Name
	awayTeam := GetTeamShortName(awayFull)
	homeTeam := GetTeamShortName(homeFull)
	awayColors := GetTeamColors(awayFull)
	homeColors := GetTeamColors(homeFull)

	// Content width = card width minus padding (1 each side)
	// Border is outside Width, already accounted for in grid calculation
	contentWidth := cardWidth - 2
	if contentWidth < 20 {
		contentWidth = 20
	}

	status := m.formatGameStatus(game)
	var lines []string

	switch game.Status.AbstractGameState {
	case "Preview":
		awayRecord := fmt.Sprintf("(%d-%d)", game.Teams.Away.LeagueRecord.Wins, game.Teams.Away.LeagueRecord.Losses)
		homeRecord := fmt.Sprintf("(%d-%d)", game.Teams.Home.LeagueRecord.Wins, game.Teams.Home.LeagueRecord.Losses)

		awayText := truncate(awayTeam+" "+awayRecord, contentWidth)
		homeText := truncate(homeTeam+" "+homeRecord, contentWidth)

		awayLine := lipgloss.NewStyle().Foreground(awayColors.Primary).Render(fmt.Sprintf("%-*s", contentWidth, awayText))
		homeLine := lipgloss.NewStyle().Foreground(homeColors.Primary).Render(fmt.Sprintf("%-*s", contentWidth, homeText))

		lines = []string{
			lipgloss.NewStyle().Width(contentWidth).Render(status),
			awayLine,
			homeLine,
		}

	case "Live", "Final":
		awayScore := game.Teams.Away.Score
		homeScore := game.Teams.Home.Score

		var awayR, awayH, awayE, homeR, homeH, homeE int
		if game.LiveData != nil {
			awayR = game.LiveData.Linescore.Teams.Away.Runs
			awayH = game.LiveData.Linescore.Teams.Away.Hits
			awayE = game.LiveData.Linescore.Teams.Away.Errors
			homeR = game.LiveData.Linescore.Teams.Home.Runs
			homeH = game.LiveData.Linescore.Teams.Home.Hits
			homeE = game.LiveData.Linescore.Teams.Home.Errors
		} else if game.Linescore != nil {
			awayR = game.Linescore.Teams.Away.Runs
			awayH = game.Linescore.Teams.Away.Hits
			awayE = game.Linescore.Teams.Away.Errors
			homeR = game.Linescore.Teams.Home.Runs
			homeH = game.Linescore.Teams.Home.Hits
			homeE = game.Linescore.Teams.Home.Errors
		} else {
			awayR = awayScore
			homeR = homeScore
		}

		awayRecord := fmt.Sprintf("(%d-%d)", game.Teams.Away.LeagueRecord.Wins, game.Teams.Away.LeagueRecord.Losses)
		homeRecord := fmt.Sprintf("(%d-%d)", game.Teams.Home.LeagueRecord.Wins, game.Teams.Home.LeagueRecord.Losses)

		awayName := truncate(awayTeam+" "+awayRecord, contentWidth-10)
		homeName := truncate(homeTeam+" "+homeRecord, contentWidth-10)

		// Right-align R H E header to match score column format " %2d %2d %2d"
		// Use lipgloss.Width for visible width since status contains ANSI escapes
		rhe := fmt.Sprintf(" %2s %2s %2s", "R", "H", "E")
		gap := contentWidth - lipgloss.Width(status) - len(rhe)
		if gap < 1 {
			gap = 1
		}
		statusLine := status + strings.Repeat(" ", gap) + rhe

		awayStyle := lipgloss.NewStyle().Foreground(awayColors.Primary)
		homeStyle := lipgloss.NewStyle().Foreground(homeColors.Primary)
		scoreStyle := lipgloss.NewStyle().Bold(true)

		if game.Status.AbstractGameState == "Final" {
			if game.Teams.Away.IsWinner {
				awayStyle = awayStyle.Bold(true)
			}
			if game.Teams.Home.IsWinner {
				homeStyle = homeStyle.Bold(true)
			}
		}

		scoreStr := fmt.Sprintf(" %2d %2d %2d", awayR, awayH, awayE)
		awayLine := awayStyle.Render(fmt.Sprintf("%-*s", contentWidth-len(scoreStr), awayName)) +
			scoreStyle.Render(scoreStr)
		scoreStr = fmt.Sprintf(" %2d %2d %2d", homeR, homeH, homeE)
		homeLine := homeStyle.Render(fmt.Sprintf("%-*s", contentWidth-len(scoreStr), homeName)) +
			scoreStyle.Render(scoreStr)

		lines = []string{statusLine, awayLine, homeLine}

	default:
		lines = []string{
			status,
			awayTeam,
			homeTeam,
		}
	}

	content := strings.Join(lines, "\n")

	cardStyle := lipgloss.NewStyle().
		Width(cardWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.AdaptiveColor{Light: "#DDDDDD", Dark: "#444444"}).
		Padding(0, 1)

	if selected {
		innerStyle := lipgloss.NewStyle().
			Width(cardWidth).
			Padding(0, 1)
		rendered := innerStyle.Render(content)
		return anim.RenderGradientBorder(rendered, cardWidth, awayColors.Primary, homeColors.Primary, lipgloss.DoubleBorder())
	}

	return cardStyle.Render(content)
}

// truncate shortens a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 0 {
		return ""
	}
	return s[:maxLen]
}

// formatGameStatus formats the game status line without emojis
func (m Model) formatGameStatus(game api.Game) string {
	switch game.Status.AbstractGameState {
	case "Preview":
		startTime := ""
		if game.GameData != nil && !game.GameData.Datetime.DateTime.IsZero() {
			startTime = game.GameData.Datetime.DateTime.Local().Format("3:04 PM MST")
		} else if !game.GameDate.IsZero() {
			startTime = game.GameDate.Local().Format("3:04 PM MST")
		} else {
			startTime = "TBD"
		}
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4A90E2")).
			Bold(true).
			Render(startTime)

	case "Live":
		// Show pre-game states (warmup, etc.) instead of inning
		if ds := game.Status.DetailedState; ds == "Warmup" || ds == "Pre-Game" {
			return lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFB86C")).
				Bold(true).
				Render(ds)
		}

		var inningStr string
		if ls := getGameLinescore(game); ls != nil {
			inningState := "Mid"
			if ls.InningState == "Top" {
				inningState = "Top"
			} else if ls.InningState == "Bottom" {
				inningState = "Bot"
			}
			inningStr = fmt.Sprintf("%s %s", inningState, ls.CurrentInningOrdinal)
		} else {
			inningStr = "In Progress"
		}
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4444")).
			Bold(true).
			Render(inningStr)

	case "Final":
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#50FA7B")).
			Render("Final")

	default:
		return previewStyle.Render(game.Status.DetailedState)
	}
}

// getScheduleGradientColors returns gradient color stops for schedule indicators.
// Uses selected game's team colors if available, config palette otherwise.
func getScheduleGradientColors(m Model) (lipgloss.Color, lipgloss.Color) {
	if len(m.games) > 0 && m.selectedGameIdx >= 0 && m.selectedGameIdx < len(m.games) {
		awayColors := GetTeamColors(m.games[m.selectedGameIdx].Teams.Away.Team.Name)
		homeColors := GetTeamColors(m.games[m.selectedGameIdx].Teams.Home.Team.Name)
		return awayColors.Primary, homeColors.Primary
	}
	return colorPrimary, colorAccent
}
