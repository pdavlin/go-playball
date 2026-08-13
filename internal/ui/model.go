package ui

import (
	"fmt"
	"image/color"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pdavlin/go-playball/internal/api"
	"github.com/pdavlin/go-playball/internal/config"
	"github.com/pdavlin/go-playball/internal/reportcache"
	"github.com/pdavlin/go-playball/internal/scouting"
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

// LiveTab represents the active card in the right column of the live
// game status view.
type LiveTab int

const (
	LiveTabPlays LiveTab = iota
	LiveTabPitchMix
	LiveTabWinProb
)

// PregameTab represents the active tab in a Preview-state game view.
// Pitchers is the zero value and the default tab on entry. The 3-column
// overview always renders above the strip, so it is not a tab.
type PregameTab int

const (
	PregameTabPitchers PregameTab = iota
	PregameTabHotBats
	PregameTabH2H
	PregameTabBullpen
)

// Model represents the main application state
type Model struct {
	view      View
	width     int
	height    int
	config    *config.Config
	apiClient *api.Client
	spinner   *anim.Spinner
	loading   bool
	err       error

	// Schedule view state
	scheduleDate         time.Time
	games                []api.Game
	selectedGameIdx      int
	scheduleScrollOffset int

	// Standings view state
	standings             []api.DivisionStandings
	wbcStandings          []api.WBCPool
	standingsScrollOffset int

	// Game view state
	currentGame        *api.Game
	gameScrollOffset   int
	gameRawJSON        []byte
	gameTimestamp      string
	gameSubview        GameSubview
	liveTab            LiveTab
	winProb            []api.WinProbPlay
	focusedPanel       int
	panelScrollOffsets [4]int

	// Pregame view state
	pregameTab     PregameTab
	pregameData    map[int]*pregameGameData
	pregameSpinner *anim.Spinner
	hotBatsWindow  HotBatsWindow

	// Score animation state
	prevAwayScore int
	prevHomeScore int
	scoreAnim     *anim.ScoreAnim

	// Track which game we expect data for, to discard stale responses
	expectedGameID int
	// If set, launch directly into this game on init
	initialGameID int

	// Report-modal state. reportModal is non-nil when the overlay is
	// active. Caches are created lazily on first open of each kind.
	reportModal   *reportModal
	scoutingCache *scouting.Cache
	recapCache    *reportcache.Cache
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

type winProbLoadedMsg struct {
	gameID int
	plays  []api.WinProbPlay
	err    error
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
		pregameData:  map[int]*pregameGameData{},
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
		// Modal owns the keyboard while open. ctrl+c still quits.
		if m.reportModal != nil {
			if msg.String() == "ctrl+c" {
				if m.reportModal.cancel != nil {
					m.reportModal.cancel()
				}
				return m, tea.Quit
			}
			closeModal, cmd := m.handleReportKey(msg)
			if closeModal {
				m.reportModal = nil
			}
			return m, cmd
		}
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
				m.winProb = nil
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
				m.winProb = nil
				m.wbcStandings = nil
				m.standingsScrollOffset = 0
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
		if m.pregameSpinner != nil {
			var cmd tea.Cmd
			m.pregameSpinner, cmd = m.pregameSpinner.Update(msg)
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
			if m.config.FocusFavoriteTeam {
				for i, g := range m.games {
					if m.config.IsFavoriteTeam(g.Teams.Away.Team.Name) ||
						m.config.IsFavoriteTeam(g.Teams.Home.Team.Name) {
						m.selectedGameIdx = i
						break
					}
				}
			}
		}
		cmds = append(cmds, scheduleTick(m.config.ScheduleRefreshSeconds))

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
			m.pregameTab = PregameTabPitchers
			m.focusedPanel = 0
			m.panelScrollOffsets = [4]int{}
			m.gameScrollOffset = 0
			if isPreviewGame(msg.game) {
				if cmd := m.dispatchPitcherDetail(); cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
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
					totalInnings := liveTotalInnings(ls)
					// Match the width the static row would use at this
					// terminal width, minus the 3-char abbreviation.
					layout := liveLinescoreLayout(totalInnings, m.width)
					lineWidth := layout.width(totalInnings) - 3
					awayText := buildScoreLineText(ls, totalInnings, "away", layout)
					homeText := buildScoreLineText(ls, totalInnings, "home", layout)
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
			isInitialLoad := m.currentGame == nil
			if isInitialLoad && msg.game.LiveData != nil {
				m.prevAwayScore = msg.game.LiveData.Linescore.Teams.Away.Runs
				m.prevHomeScore = msg.game.LiveData.Linescore.Teams.Home.Runs
			}
			m.currentGame = msg.game
			m.gameRawJSON = msg.rawJSON
			m.gameTimestamp = msg.timestamp

			// On entry into a Preview game, default to the Pitchers
			// tab and kick off the arsenal + recent-starts fetch so
			// content appears without an extra keypress.
			if isInitialLoad && isPreviewGame(m.currentGame) {
				m.pregameTab = PregameTabPitchers
				if cmd := m.dispatchPitcherDetail(); cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}
		if m.currentGame != nil && isGameLive(m.currentGame) {
			wait := 10 * time.Second
			if m.currentGame.MetaData != nil && m.currentGame.MetaData.Wait > 0 {
				wait = time.Duration(m.currentGame.MetaData.Wait) * time.Second
			}
			cmds = append(cmds, scheduleGameUpdateIncremental(
				m.currentGame.ID, m.gameRawJSON, m.gameTimestamp, wait))
			cmds = append(cmds, loadWinProbability(m.apiClient, m.currentGame.ID))
		}

	case gameTickMsg:
		if m.view == GameView && msg.gameID == m.expectedGameID {
			// Refresh quietly: no loading flag, no Updating spinner.
			// The spinner prefix lengthened the help bar and could wrap
			// it to a second line, shifting the whole layout every tick.
			cmds = append(cmds, loadGameIncremental(m.apiClient, msg.gameID, msg.rawJSON, msg.timestamp))
			return m, tea.Batch(cmds...)
		}

	case winProbLoadedMsg:
		if msg.gameID == m.expectedGameID && msg.err == nil {
			m.winProb = msg.plays
		}

	case pregameDataLoadedMsg:
		if m.pregameData == nil {
			m.pregameData = map[int]*pregameGameData{}
		}
		data, ok := m.pregameData[msg.gameID]
		if !ok {
			data = &pregameGameData{}
			m.pregameData[msg.gameID] = data
		}
		switch p := msg.payload.(type) {
		case pitcherDetailPayload:
			data.pitcherDetail = &p
		case hotBatsLoadedPayload:
			if data.hotBats == nil {
				data.hotBats = &hotBatsPayload{
					awayByWindow: map[HotBatsWindow]*hotBatsTeamPayload{},
					homeByWindow: map[HotBatsWindow]*hotBatsTeamPayload{},
				}
			}
			away := p.away
			home := p.home
			data.hotBats.awayByWindow[p.window] = &away
			data.hotBats.homeByWindow[p.window] = &home
		case h2hPayload:
			data.h2h = &p
		case bullpenPayload:
			data.bullpen = &p
		}
		if m.pregameSpinner != nil {
			m.pregameSpinner = m.pregameSpinner.Pause()
		}

	case tickMsg:
		// Re-fetch schedule if on schedule view
		if m.view == ScheduleView {
			cmds = append(cmds, loadSchedule(m.apiClient, m.scheduleDate))
		}

	case reportEventMsg:
		if m.reportModal != nil && m.reportModal.gamePk == msg.gamePk {
			m.reportModal.applyEvent(msg.ev)
			if !m.reportModal.streamDone && m.reportModal.nextEvent != nil {
				cmds = append(cmds, m.reportModal.nextEvent())
			}
		}

	case reportClosedMsg:
		if m.reportModal != nil && m.reportModal.gamePk == msg.gamePk {
			// Spinner stays running; advanceReveal pauses it once the
			// reveal cursor catches up with the streamed text.
			m.reportModal.streamDone = true
		}
	}

	// Keep the modal spinner ticking while open and streaming. Each
	// consumed tick also advances the per-character reveal cursor.
	if m.reportModal != nil && m.reportModal.spinner != nil {
		if tickMsg, ok := msg.(anim.SpinnerTickMsg); ok {
			var cmd tea.Cmd
			m.reportModal.spinner, cmd = m.reportModal.spinner.Update(tickMsg)
			if cmd != nil {
				m.reportModal.advanceReveal()
			}
			cmds = append(cmds, cmd)
		}
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

	rendered := fmt.Sprintf("%s\n%s", content, helpBar)

	if m.reportModal != nil {
		rendered = m.renderReportModal(rendered)
	}

	return rendered
}

// helpItem is one whole entry in the help bar (e.g. "q: quit"). Entries
// are never cut mid-token: buildHelpBar either renders an item in full
// or drops it entirely.
type helpItem struct {
	text string
	// priority ranks which items survive when the bar is too narrow
	// for all of them. Higher priority is kept longest. Use the
	// helpPriority* constants below rather than raw numbers.
	priority int
}

// Priority tiers for help-bar items, from "drop first" to "never drop".
// Every per-view item list below is commented with which tier its
// entries landed in and why.
const (
	helpPriorityConvenience = iota // nice-to-have shortcuts (day paging, panel numbers, window toggle)
	helpPriorityNav                // secondary navigation (view/tab switches, scroll)
	helpPriorityCore               // the view's primary action and global quit; never drop while anything fits
)

// buildHelpBar assembles a single-line help string from items, keeping
// as many WHOLE items as fit within width. Items are considered in
// descending priority order so a wide low-priority item never bumps a
// higher-priority item off the bar just because it sits later in the
// list; survivors are then joined in their original left-to-right
// order (with " | " separators), so the bar reads the same as before,
// just possibly shorter.
func buildHelpBar(items []helpItem, width int) string {
	if width <= 0 || len(items) == 0 {
		return ""
	}

	order := make([]int, len(items))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		return items[order[a]].priority > items[order[b]].priority
	})

	kept := make([]bool, len(items))
	total := 0
	count := 0
	for _, idx := range order {
		w := lipgloss.Width(items[idx].text)
		sep := 0
		if count > 0 {
			sep = 3 // " | "
		}
		if total+sep+w > width {
			continue
		}
		total += sep + w
		count++
		kept[idx] = true
	}

	parts := make([]string, 0, count)
	for i, it := range items {
		if kept[i] {
			parts = append(parts, it.text)
		}
	}
	return strings.Join(parts, " | ")
}

// reportHelpEntry returns the help-bar fragment for the current schedule
// selection ("r: scouting", "r: recap") or "" when no report is
// available.
func (m Model) reportHelpEntry() string {
	if m.selectedGameIdx < 0 || m.selectedGameIdx >= len(m.games) {
		return ""
	}
	_, label, ok := reportKindFor(m.config.ScoutingEnabled(), &m.games[m.selectedGameIdx])
	if !ok {
		return ""
	}
	return "r: " + label
}

// activeSubviewLabel returns the help item text for the currently
// active game subview, used to apply the team-color gradient. Empty
// when there is no such label (non-game views).
func (m Model) activeSubviewLabel() string {
	switch m.gameSubview {
	case BoxScoreSubview:
		return "b: box score"
	case AllPlaysSubview:
		return "a: all plays"
	case ScoringPlaysSubview:
		return "p: scoring"
	case GameStatusSubview:
		return "g: game"
	}
	return ""
}

// helpItems returns the full, priority-tagged item list for the
// current view/subview state. Order is the desired left-to-right
// display order; buildHelpBar decides which survive at the current
// width.
func (m Model) helpItems() []helpItem {
	quit := helpItem{"q: quit", helpPriorityCore}
	schedule := helpItem{"c: schedule", helpPriorityNav}
	standings := helpItem{"s: standings", helpPriorityNav}

	switch m.view {
	case ScheduleView:
		items := []helpItem{
			schedule,
			standings,
			{"hjkl/arrows: navigate", helpPriorityCore},
			{"enter: view game", helpPriorityCore},
			{"p/n: prev/next day", helpPriorityConvenience},
			{"t: today", helpPriorityConvenience},
			quit,
		}
		if entry := m.reportHelpEntry(); entry != "" {
			items = append([]helpItem{{entry, helpPriorityNav}}, items...)
		}
		return items

	case StandingsView:
		return []helpItem{schedule, standings, {"jk: scroll", helpPriorityNav}, quit}

	case GameView:
		base := []helpItem{schedule, standings, quit}
		return append(m.gameSubviewHelpItems(), base...)
	}

	return nil
}

// gameSubviewHelpItems returns the subview-specific items for the game
// view (everything before the shared c/s/q base). It is state-aware so
// it never advertises a key that does nothing in the current state:
//   - Final games never show "1-3/hl: tabs" (the live tab strip):
//     renderFinalGame ignores m.liveTab entirely, so those keys are
//     dead there. Panel selection (1-4) is also only wired up in
//     BoxScoreSubview (handleGameStatusKeys, reached via GameStatusSubview,
//     doesn't route 1-4 to handleBoxScoreKeys), so it's omitted from the
//     GameStatusSubview branch even for a final game.
//   - Preview games drop "a: all plays" and "p: scoring": renderGame's
//     Preview branch always renders the pregame tabs regardless of
//     gameSubview, so those keys have no visible effect before the game
//     starts. "b: box score" is dropped for the same reason -- the box
//     score never renders pre-game -- and because switching gameSubview
//     to BoxScoreSubview would silently reroute jk from tab-scroll to
//     panel-scroll on the next keypress.
func (m Model) gameSubviewHelpItems() []helpItem {
	g := helpItem{"g: game", helpPriorityNav}
	b := helpItem{"b: box score", helpPriorityNav}
	a := helpItem{"a: all plays", helpPriorityNav}
	p := helpItem{"p: scoring", helpPriorityNav}
	scroll := helpItem{"jk: scroll", helpPriorityNav}

	// Whichever subview is active gets bumped to core priority so it's
	// the last thing dropped -- you should always be able to tell what
	// key got you to the screen you're looking at.
	bump := func(it helpItem, label string) helpItem {
		if label == it.text {
			it.priority = helpPriorityCore
		}
		return it
	}
	active := m.activeSubviewLabel()
	g, b, a, p = bump(g, active), bump(b, active), bump(a, active), bump(p, active)

	switch m.gameSubview {
	case BoxScoreSubview:
		return []helpItem{g, b, a, p, {"1-4: panels", helpPriorityConvenience}, scroll}
	case AllPlaysSubview, ScoringPlaysSubview:
		return []helpItem{g, b, a, p, scroll}
	case GameStatusSubview:
		if isPreviewGame(m.currentGame) {
			items := []helpItem{
				{"1-4/hl: tabs", helpPriorityCore},
				scroll,
			}
			if m.pregameTab == PregameTabHotBats {
				items = append(items, helpItem{", .: window", helpPriorityConvenience})
			}
			return items
		}
		if isFinalGame(m.currentGame) {
			// GameStatusSubview on a final game renders the same box
			// score as BoxScoreSubview, and handleGameKeys routes 1-4
			// and j/k to the panels there, so advertise them.
			return []helpItem{g, b, a, p, {"1-4: panels", helpPriorityConvenience}, scroll}
		}
		return []helpItem{{"1-3/hl: tabs", helpPriorityCore}, scroll, b, a, p}
	}
	return []helpItem{scroll}
}

// renderHelpBar renders the help bar with keyboard shortcuts, fit to
// the terminal width as whole items (see buildHelpBar).
func (m Model) renderHelpBar() string {
	// helpStyle carries Padding(0, 1): 1 column on each side counts
	// against the content budget below.
	budget := m.width - 2
	if budget < 0 {
		budget = 0
	}

	spinnerPrefix := ""
	if m.spinner != nil && m.spinner.State() == anim.SpinnerRunning {
		spinnerPrefix = m.spinner.View() + "  "
		budget -= lipgloss.Width(spinnerPrefix)
		if budget < 0 {
			budget = 0
		}
	}

	help := buildHelpBar(m.helpItems(), budget)

	// Apply team-color gradient to the active subview label, if it
	// survived the width cut.
	if m.view == GameView && m.currentGame != nil {
		if activeLabel := m.activeSubviewLabel(); activeLabel != "" && strings.Contains(help, activeLabel) {
			away, home := getGameTeamColors(m.currentGame)
			gradLabel := anim.BlendGradientBold(activeLabel, away, home)
			help = strings.Replace(help, activeLabel, gradLabel, 1)
		}
	}

	help = spinnerPrefix + help

	// MaxHeight(1) stays as a final guard: buildHelpBar already fits
	// the content to width, but this keeps the bar from ever growing
	// past one line if that invariant is violated some other way.
	return helpStyle.Width(m.width).MaxHeight(1).Render(help)
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

func loadWinProbability(client *api.Client, gameID int) tea.Cmd {
	return func() tea.Msg {
		plays, err := client.FetchWinProbability(gameID)
		return winProbLoadedMsg{gameID: gameID, plays: plays, err: err}
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

func scheduleTick(seconds int) tea.Cmd {
	if seconds < 5 {
		seconds = 5
	}
	return tea.Tick(time.Duration(seconds)*time.Second, func(t time.Time) tea.Msg {
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
