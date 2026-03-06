package ui

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pdavlin/go-playball/internal/api"
	"github.com/pdavlin/go-playball/internal/config"
	"github.com/pdavlin/go-playball/internal/ui/anim"
)

// View represents the current active view
type View int

const (
	ScheduleView View = iota
	StandingsView
	GameView
)

// GameSubview represents the active sub-tab within the game view
type GameSubview int

const (
	GameStatusSubview GameSubview = iota
	BoxScoreSubview
	AllPlaysSubview
	ScoringPlaysSubview
)

// Model represents the main application state
type Model struct {
	view          View
	width         int
	height        int
	config        *config.Config
	apiClient     *api.Client
	spinner       *anim.Spinner
	loading       bool
	err           error

	// Schedule view state
	scheduleDate       time.Time
	games              []api.Game
	selectedGameIdx    int
	scheduleScrollOffset int

	// Standings view state
	standings        []api.DivisionStandings
	wbcStandings     []api.WBCPool

	// Game view state
	currentGame        *api.Game
	gameScrollOffset   int
	gameRawJSON        []byte
	gameTimestamp      string
	gameSubview        GameSubview
	focusedPanel       int
	panelScrollOffsets [4]int

	// Score animation state
	prevAwayScore int
	prevHomeScore int
	scoreAnim     *anim.ScoreAnim

	// Track which game we expect data for, to discard stale responses
	expectedGameID int
	// If set, launch directly into this game on init
	initialGameID int
}

// Message types for async operations
type scheduleLoadedMsg struct {
	games []api.Game
	err   error
}

type standingsLoadedMsg struct {
	standings []api.DivisionStandings
	err       error
}

type wbcStandingsLoadedMsg struct {
	pools []api.WBCPool
	err   error
}

type gameLoadedMsg struct {
	game *api.Game
	err  error
}

type tickMsg time.Time

type gameIncrementalLoadedMsg struct {
	gameID    int
	game      *api.Game
	rawJSON   []byte
	timestamp string
	err       error
}

type gameTickMsg struct {
	gameID    int
	rawJSON   []byte
	timestamp string
}

// NewModel creates a new application model.
// If initialGameID > 0, the TUI launches directly into that game.
func NewModel(cfg *config.Config, initialGameID int) Model {
	// Detect terminal background for color adjustments
	DetectDarkMode(lipgloss.HasDarkBackground())

	// Update colors from config
	UpdateColors(
		cfg.Colors.Primary,
		cfg.Colors.Secondary,
		cfg.Colors.Accent,
		cfg.Colors.Error,
		cfg.Colors.Success,
	)
	UpdateEventColors(cfg.EventColors)

	// Create initial spinner for schedule load
	s := anim.NewSpinner(15, "Loading", colorPrimary, colorAccent)
	s, _ = s.Start()

	m := Model{
		view:         ScheduleView,
		config:       cfg,
		apiClient:    api.NewClient(),
		spinner:      s,
		loading:      true,
		scheduleDate: time.Now(),
		games:        []api.Game{},
		standings:    []api.DivisionStandings{},
	}

	if initialGameID > 0 {
		m.view = GameView
		m.expectedGameID = initialGameID
		m.initialGameID = initialGameID
	}

	return m
}

// startSpinner creates a new spinner and returns its first tick command.
func (m *Model) startSpinner(label string, from, to color.Color) tea.Cmd {
	m.spinner = anim.NewSpinner(15, label, from, to)
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Start()
	return cmd
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	if m.spinner != nil {
		cmds = append(cmds, m.spinner.Tick())
	}
	if m.initialGameID > 0 {
		cmds = append(cmds, loadGameIncremental(m.apiClient, m.initialGameID, nil, ""))
	} else {
		cmds = append(cmds, loadSchedule(m.apiClient, m.scheduleDate))
	}
	return tea.Batch(cmds...)
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "c":
			if m.view != ScheduleView {
				m.view = ScheduleView
				m.currentGame = nil
				m.expectedGameID = 0
				m.gameRawJSON = nil
				m.gameTimestamp = ""
				m.scoreAnim = nil
				m.loading = true
				spinnerCmd := m.startSpinner("Loading", colorPrimary, colorAccent)
				return m, tea.Batch(spinnerCmd, loadSchedule(m.apiClient, m.scheduleDate))
			}
		case "s":
			if m.view != StandingsView {
				m.view = StandingsView
				m.currentGame = nil
				m.expectedGameID = 0
				m.gameRawJSON = nil
				m.gameTimestamp = ""
				m.scoreAnim = nil
				m.wbcStandings = nil
				m.loading = true
				spinnerCmd := m.startSpinner("Loading", colorPrimary, colorAccent)
				cmds := []tea.Cmd{spinnerCmd, loadStandings(m.apiClient)}
				if hasWBCGames(m.games) {
					cmds = append(cmds, loadWBCStandings(m.apiClient, m.scheduleDate))
				}
				return m, tea.Batch(cmds...)
			}
		}

		// View-specific key handling
		switch m.view {
		case ScheduleView:
			return m.handleScheduleKeys(msg)
		case StandingsView:
			return m.handleStandingsKeys(msg)
		case GameView:
			return m.handleGameKeys(msg)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case anim.SpinnerTickMsg:
		if m.spinner != nil {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case anim.ScoreAnimTickMsg:
		if m.scoreAnim != nil {
			var cmd tea.Cmd
			m.scoreAnim, cmd = m.scoreAnim.Update(msg)
			cmds = append(cmds, cmd)
		}

	case scheduleLoadedMsg:
		m.loading = false
		if m.spinner != nil {
			m.spinner = m.spinner.Pause()
		}
		m.err = msg.err
		if msg.err == nil {
			wbc, mlb := partitionGames(msg.games)
			sortGames(wbc)
			sortGames(mlb)
			m.games = append(wbc, mlb...)
			if len(m.games) > 0 && m.selectedGameIdx >= len(m.games) {
				m.selectedGameIdx = len(m.games) - 1
			}
		}

	case standingsLoadedMsg:
		m.loading = false
		if m.spinner != nil {
			m.spinner = m.spinner.Pause()
		}
		m.err = msg.err
		if msg.err == nil {
			m.standings = msg.standings
		}

	case wbcStandingsLoadedMsg:
		if msg.err == nil {
			m.wbcStandings = msg.pools
		}

	case gameLoadedMsg:
		m.loading = false
		if m.spinner != nil {
			m.spinner = m.spinner.Pause()
		}
		m.err = msg.err
		if msg.err == nil {
			m.currentGame = msg.game
			// Set default subview based on game state
			gameState := ""
			if msg.game.GameData != nil {
				gameState = msg.game.GameData.Status.AbstractGameState
			}
			if gameState == "Final" {
				m.gameSubview = BoxScoreSubview
			} else {
				m.gameSubview = GameStatusSubview
			}
			m.focusedPanel = 0
			m.panelScrollOffsets = [4]int{}
			m.gameScrollOffset = 0
			// Initialize score tracking for incremental updates
			if msg.game.LiveData != nil {
				m.prevAwayScore = msg.game.LiveData.Linescore.Teams.Away.Runs
				m.prevHomeScore = msg.game.LiveData.Linescore.Teams.Home.Runs
			}
			m.scoreAnim = nil
		}

	case gameIncrementalLoadedMsg:
		// Discard responses for a game we're no longer viewing
		if msg.gameID != m.expectedGameID {
			return m, nil
		}
		m.loading = false
		if m.spinner != nil {
			m.spinner = m.spinner.Pause()
		}
		m.err = msg.err
		if msg.err == nil {
			// Detect score changes, but only on incremental updates (not initial load)
			if m.currentGame != nil && msg.game.LiveData != nil {
				newAway := msg.game.LiveData.Linescore.Teams.Away.Runs
				newHome := msg.game.LiveData.Linescore.Teams.Home.Runs
				if newAway != m.prevAwayScore || newHome != m.prevHomeScore {
					awayC, homeC := getGameTeamColorsFull(msg.game)
					ls := msg.game.LiveData.Linescore
					totalInnings := ls.CurrentInning
					if totalInnings < 9 {
						totalInnings = 9
					}
					// Width: innings (3 chars each) + gap (2) + R (2) + H E (6)
					lineWidth := totalInnings*3 + 10
					awayText := buildScoreLineText(ls, totalInnings, "away")
					homeText := buildScoreLineText(ls, totalInnings, "home")
					sa := anim.NewScoreAnim(
						newAway != m.prevAwayScore,
						newHome != m.prevHomeScore,
						lineWidth,
						awayC.Primary, awayC.Secondary,
						homeC.Primary, homeC.Secondary,
						awayText, homeText,
					)
					var cmd tea.Cmd
					m.scoreAnim, cmd = sa.Start()
					cmds = append(cmds, cmd)
					m.prevAwayScore = newAway
					m.prevHomeScore = newHome
				}
			}
			// Seed score tracking on initial load
			if m.currentGame == nil && msg.game.LiveData != nil {
				m.prevAwayScore = msg.game.LiveData.Linescore.Teams.Away.Runs
				m.prevHomeScore = msg.game.LiveData.Linescore.Teams.Home.Runs
			}
			m.currentGame = msg.game
			m.gameRawJSON = msg.rawJSON
			m.gameTimestamp = msg.timestamp
		}
		if m.currentGame != nil && isGameLive(m.currentGame) {
			wait := 10 * time.Second
			if m.currentGame.MetaData != nil && m.currentGame.MetaData.Wait > 0 {
				wait = time.Duration(m.currentGame.MetaData.Wait) * time.Second
			}
			cmds = append(cmds, scheduleGameUpdateIncremental(
				m.currentGame.ID, m.gameRawJSON, m.gameTimestamp, wait))
		}

	case gameTickMsg:
		if m.view == GameView && msg.gameID == m.expectedGameID {
			m.loading = true
			if m.currentGame != nil {
				away, home := getGameTeamColors(m.currentGame)
				cmds = append(cmds, m.startSpinner("Updating", away, home))
			}
			cmds = append(cmds, loadGameIncremental(m.apiClient, msg.gameID, msg.rawJSON, msg.timestamp))
			return m, tea.Batch(cmds...)
		}

	case tickMsg:
		// Re-fetch schedule if on schedule view
		if m.view == ScheduleView {
			cmds = append(cmds, loadSchedule(m.apiClient, m.scheduleDate))
		}
		// Schedule next tick
		cmds = append(cmds, tick())
	}

	return m, tea.Batch(cmds...)
}

// View renders the current view
func (m Model) View() string {
	if m.width == 0 {
		return ""
	}

	var content string
	switch m.view {
	case ScheduleView:
		content = m.renderSchedule()
	case StandingsView:
		content = m.renderStandings()
	case GameView:
		content = m.renderGame()
	}

	// Add help bar at bottom
	helpBar := m.renderHelpBar()

	// Calculate how many lines we have
	contentLines := strings.Count(content, "\n") + 1
	helpLines := 1 // Help bar is one line
	usedLines := contentLines + helpLines

	// Fill remaining vertical space to make it full-screen
	// Reserve height for help bar
	if m.height > 0 && usedLines < m.height {
		paddingLines := m.height - usedLines - 1
		if paddingLines > 0 {
			content += strings.Repeat("\n", paddingLines)
		}
	}

	return fmt.Sprintf("%s\n%s", content, helpBar)
}

// renderHelpBar renders the help bar with keyboard shortcuts
func (m Model) renderHelpBar() string {
	var help string
	switch m.view {
	case ScheduleView:
		help = "c: schedule | s: standings | hjkl/arrows: navigate | enter: view game | p/n: prev/next day | t: today | q: quit"
	case StandingsView:
		help = "c: schedule | s: standings | q: quit"
	case GameView:
		base := "c: schedule | s: standings | q: quit"

		switch m.gameSubview {
		case BoxScoreSubview:
			help = "g: game | b: box score | a: all plays | p: scoring | 1-4: panels | jk: scroll | " + base
		case AllPlaysSubview:
			help = "g: game | b: box score | a: all plays | p: scoring | jk: scroll | " + base
		case ScoringPlaysSubview:
			help = "g: game | b: box score | a: all plays | p: scoring | jk: scroll | " + base
		case GameStatusSubview:
			help = "b: box score | a: all plays | p: scoring | jk: scroll | " + base
		default:
			help = "jk: scroll | " + base
		}
	}

	// Apply team-color gradient to the active subview label
	if m.view == GameView && m.currentGame != nil {
		activeLabel := ""
		switch m.gameSubview {
		case BoxScoreSubview:
			activeLabel = "b: box score"
		case AllPlaysSubview:
			activeLabel = "a: all plays"
		case ScoringPlaysSubview:
			activeLabel = "p: scoring"
		case GameStatusSubview:
			activeLabel = "g: game"
		}
		if activeLabel != "" {
			away, home := getGameTeamColors(m.currentGame)
			gradLabel := anim.BlendGradientBold(activeLabel, away, home)
			help = strings.Replace(help, activeLabel, gradLabel, 1)
		}
	}

	if m.spinner != nil && m.spinner.State() == anim.SpinnerRunning {
		help = m.spinner.View() + "  " + help
	}

	return helpStyle.Width(m.width).Render(help)
}

// Command functions for async operations

func loadSchedule(client *api.Client, date time.Time) tea.Cmd {
	return func() tea.Msg {
		games, err := client.FetchSchedule(date, "1,51")
		return scheduleLoadedMsg{games: games, err: err}
	}
}

func loadStandings(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		standings, err := client.FetchStandings()
		return standingsLoadedMsg{standings: standings, err: err}
	}
}

func loadGame(client *api.Client, gameID int) tea.Cmd {
	return func() tea.Msg {
		game, err := client.FetchGame(gameID)
		return gameLoadedMsg{game: game, err: err}
	}
}

func loadGameIncremental(client *api.Client, gameID int, currentJSON []byte, timestamp string) tea.Cmd {
	return func() tea.Msg {
		game, rawJSON, err := client.FetchGameIncremental(gameID, currentJSON, timestamp)
		if err != nil {
			return gameIncrementalLoadedMsg{gameID: gameID, err: err}
		}
		ts := ""
		if game.MetaData != nil {
			ts = game.MetaData.TimeStamp
		}
		return gameIncrementalLoadedMsg{
			gameID:    gameID,
			game:      game,
			rawJSON:   rawJSON,
			timestamp: ts,
		}
	}
}

func scheduleGameUpdateIncremental(gameID int, rawJSON []byte, timestamp string, wait time.Duration) tea.Cmd {
	return tea.Tick(wait, func(t time.Time) tea.Msg {
		return gameTickMsg{
			gameID:    gameID,
			rawJSON:   rawJSON,
			timestamp: timestamp,
		}
	})
}

func tick() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func hasWBCGames(games []api.Game) bool {
	for _, g := range games {
		if g.GameType == "F" {
			return true
		}
	}
	return false
}

func loadWBCStandings(client *api.Client, date time.Time) tea.Cmd {
	return func() tea.Msg {
		pools, err := client.FetchWBCPoolStandings(date.Year())
		return wbcStandingsLoadedMsg{pools: pools, err: err}
	}
}
