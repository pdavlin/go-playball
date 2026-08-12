package ui

import (
	"sort"
	"strconv"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pdavlin/go-playball/internal/api"
	"github.com/pdavlin/go-playball/internal/ui/anim"
)

// pregameSpinnerSize sets the width of cycling chars in skeleton cells.
// The widest skeleton cell is the Pitch column (18 visual cells); the
// spinner is sized to fully populate it.
const pregameSpinnerSize = 18

// pregameGameData is the per-game cache of fetched tab payloads.
// One entry exists per gameID a user has visited in the current session.
type pregameGameData struct {
	pitcherDetail *pitcherDetailPayload
	hotBats       *hotBatsPayload
	h2h           *h2hPayload
	bullpen       *bullpenPayload
}

// Bullpen status heuristic constants. Exported as package-level so the
// renderer footer and the heuristic itself stay in sync.
const (
	bullpenUnavailPitchThreshold = 30 // pitches >= this 0-1 days ago → Unavail
	bullpenRecentDaysWindow      = 1  // days-ago threshold for Limited vs Ready
	bullpenRecentAppearances     = 3  // sub-rows displayed per reliever
)

// bullpenStatusKind enumerates the three availability buckets.
type bullpenStatusKind int

const (
	bullpenReady bullpenStatusKind = iota
	bullpenLimited
	bullpenUnavail
)

// bullpenPayload is the per-game bullpen cache for both teams.
type bullpenPayload struct {
	away *bullpenTeamPayload
	home *bullpenTeamPayload
}

// bullpenTeamPayload is one team's reliever list with statuses.
type bullpenTeamPayload struct {
	relievers []bullpenReliever
	err       error
	loaded    bool
}

// bullpenReliever pairs a pitcher with their last few appearances and
// computed status.
type bullpenReliever struct {
	name        string
	status      bullpenStatusKind
	appearances []bullpenAppearance // newest-first, up to bullpenRecentAppearances
}

type bullpenAppearance struct {
	date           string
	inningsPitched string
	pitches        int
}

// h2hPayload is the aggregated head-to-head series for the current game.
// All counts are oriented to the CURRENT game's away/home teams; past
// meetings where the home/away assignment was swapped are normalized
// during aggregation.
type h2hPayload struct {
	games         int
	awayWins      int
	homeWins      int
	awayRunsTotal int
	homeRunsTotal int
	oneRunGames   int
	largestMargin int
	lastMeeting   *h2hMeeting
	err           error
	loaded        bool
}

// h2hMeeting represents one past meeting normalized to the current
// game's perspective.
type h2hMeeting struct {
	date     string // YYYY-MM-DD
	awayRuns int    // runs by the team that is AWAY in the current game
	homeRuns int    // runs by the team that is HOME in the current game
}

// HotBatsWindow selects the rolling-game window used for ranking.
type HotBatsWindow int

const (
	HotBatsL7 HotBatsWindow = iota
	HotBatsL15
	HotBatsL30
)

// gamesInWindow returns the number of games for the windowed lastXGames
// API call. Keep in sync with the PA thresholds in paMinForWindow.
func (w HotBatsWindow) gamesInWindow() int {
	switch w {
	case HotBatsL15:
		return 15
	case HotBatsL30:
		return 30
	default:
		return 7
	}
}

// label returns the chip text for the window selector header.
func (w HotBatsWindow) label() string {
	switch w {
	case HotBatsL15:
		return "L15"
	case HotBatsL30:
		return "L30"
	default:
		return "L7"
	}
}

// paMinForWindow returns the minimum plate appearances required to be
// eligible for ranking in the given window. Below threshold = excluded.
func paMinForWindow(w HotBatsWindow) int {
	switch w {
	case HotBatsL15:
		return 35
	case HotBatsL30:
		return 70
	default:
		return 15
	}
}

// hotBatsPayload caches per-window top-hitter tables for both teams.
// Each window is loaded independently on first activation.
type hotBatsPayload struct {
	awayByWindow map[HotBatsWindow]*hotBatsTeamPayload
	homeByWindow map[HotBatsWindow]*hotBatsTeamPayload
}

// hotBatsTeamPayload is one team's ranked top-hitter list for one window.
// loaded gates duplicate fetches; an empty rows slice with loaded=true
// is a valid "insufficient data" state.
type hotBatsTeamPayload struct {
	rows   []hotBatsRow
	err    error
	loaded bool
}

// hotBatsRow is a single hitter line displayed in the table.
type hotBatsRow struct {
	name     string
	position string
	avg      string
	homeRuns int
	rbi      int
	ops      string
	pa       int
}

// hotBatsLoadedPayload is the typed payload inside pregameDataLoadedMsg
// for Hot Bats fetches.
type hotBatsLoadedPayload struct {
	window HotBatsWindow
	away   hotBatsTeamPayload
	home   hotBatsTeamPayload
}

// pitcherDetailPayload caches the result of fetching arsenal + recent
// starts for both probable pitchers. loaded gates duplicate fetches.
type pitcherDetailPayload struct {
	awayArsenal []api.ArsenalPitch
	awayLog     []api.GameLogStart
	awayErr     error
	homeArsenal []api.ArsenalPitch
	homeLog     []api.GameLogStart
	homeErr     error
	loaded      bool
}

// pregameDataLoadedMsg is dispatched when an async pregame fetch returns.
// payload is type-switched on in the Update loop.
type pregameDataLoadedMsg struct {
	gameID  int
	tab     PregameTab
	payload any
}

// ensurePregameData returns the cache entry for gameID, creating one if
// missing. Safe to call before the map is initialized.
func (m *Model) ensurePregameData(gameID int) *pregameGameData {
	if m.pregameData == nil {
		m.pregameData = map[int]*pregameGameData{}
	}
	if _, ok := m.pregameData[gameID]; !ok {
		m.pregameData[gameID] = &pregameGameData{}
	}
	return m.pregameData[gameID]
}

// startPregameSpinner creates a fresh spinner tinted with the game's
// team colors and starts it. Replaces any prior pregame spinner.
// Returns the first tick command, or nil if the spinner can't be set up.
func (m *Model) startPregameSpinner() tea.Cmd {
	awayC, homeC := getGameTeamColors(m.currentGame)
	m.pregameSpinner = anim.NewCyclingSpinner(pregameSpinnerSize, "", awayC, homeC)
	var cmd tea.Cmd
	m.pregameSpinner, cmd = m.pregameSpinner.Start()
	return cmd
}

// dispatchPitcherDetail wraps fetchPitcherDetailIfNeeded so that the
// skeleton spinner is started in the same step as the fetch. Returns
// nil when there is nothing to do (cache hit or in-flight).
func (m *Model) dispatchPitcherDetail() tea.Cmd {
	fetchCmd := m.fetchPitcherDetailIfNeeded()
	if fetchCmd == nil {
		return nil
	}
	spinnerCmd := m.startPregameSpinner()
	return tea.Batch(spinnerCmd, fetchCmd)
}

// dispatchHotBats kicks off (or reuses) the hot-bats fetch for the
// active window. Returns nil when the window is already cached or
// in-flight for this game.
func (m *Model) dispatchHotBats(window HotBatsWindow) tea.Cmd {
	fetchCmd := m.fetchHotBatsIfNeeded(window)
	if fetchCmd == nil {
		return nil
	}
	spinnerCmd := m.startPregameSpinner()
	return tea.Batch(spinnerCmd, fetchCmd)
}

// fetchHotBatsIfNeeded reserves the (away, home) slots for `window` and
// returns a tea.Cmd that fans out the roster + per-hitter stat fetches.
// Returns nil when both teams have already been fetched for this window.
func (m Model) fetchHotBatsIfNeeded(window HotBatsWindow) tea.Cmd {
	game := m.currentGame
	if game == nil || game.GameData == nil {
		return nil
	}
	data := m.ensureHotBatsCache(game.ID)
	if data.awayByWindow[window] != nil && data.homeByWindow[window] != nil {
		return nil
	}
	// Reserve slots so a second activation doesn't dispatch again.
	if data.awayByWindow[window] == nil {
		data.awayByWindow[window] = &hotBatsTeamPayload{}
	}
	if data.homeByWindow[window] == nil {
		data.homeByWindow[window] = &hotBatsTeamPayload{}
	}

	gameID := game.ID
	season := time.Now().Year()
	awayTeamID := game.GameData.Teams.Away.ID
	homeTeamID := game.GameData.Teams.Home.ID
	games := window.gamesInWindow()
	paMin := paMinForWindow(window)
	client := m.apiClient

	return func() tea.Msg {
		var away, home hotBatsTeamPayload
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			away = computeHotBatsTeam(client, awayTeamID, season, games, paMin)
		}()
		go func() {
			defer wg.Done()
			home = computeHotBatsTeam(client, homeTeamID, season, games, paMin)
		}()
		wg.Wait()
		return pregameDataLoadedMsg{
			gameID:  gameID,
			tab:     PregameTabHotBats,
			payload: hotBatsLoadedPayload{window: window, away: away, home: home},
		}
	}
}

// ensureHotBatsCache returns the per-game hot-bats payload, creating
// the slot and inner maps if missing.
func (m *Model) ensureHotBatsCache(gameID int) *hotBatsPayload {
	data := m.ensurePregameData(gameID)
	if data.hotBats == nil {
		data.hotBats = &hotBatsPayload{
			awayByWindow: map[HotBatsWindow]*hotBatsTeamPayload{},
			homeByWindow: map[HotBatsWindow]*hotBatsTeamPayload{},
		}
	}
	return data.hotBats
}

// computeHotBatsTeam fetches the team roster + per-hitter window stats
// and returns the top-3 ranked hitters. Errors mid-flight are returned
// in the payload's err field rather than dropping the team entirely.
func computeHotBatsTeam(client *api.Client, teamID, season, games, paMin int) hotBatsTeamPayload {
	roster, err := client.FetchTeamHitters(teamID)
	if err != nil {
		return hotBatsTeamPayload{err: err, loaded: true}
	}
	if len(roster) == 0 {
		return hotBatsTeamPayload{loaded: true}
	}

	// Fan out per-hitter window fetches.
	type result struct {
		player api.RosterPlayer
		stats  *api.BatterWindowStats
	}
	results := make([]result, len(roster))
	var wg sync.WaitGroup
	for i := range roster {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			line, ferr := client.FetchHitterLastXGames(roster[i].ID, season, games)
			if ferr != nil || line == nil {
				return
			}
			results[i] = result{player: roster[i], stats: line}
		}(i)
	}
	wg.Wait()

	// Filter, rank, take top 3.
	var rows []hotBatsRow
	for _, r := range results {
		if r.stats == nil {
			continue
		}
		if r.stats.PlateAppearances < paMin {
			continue
		}
		rows = append(rows, hotBatsRow{
			name:     r.player.FullName,
			position: r.player.Position,
			avg:      r.stats.AVG,
			homeRuns: r.stats.HomeRuns,
			rbi:      r.stats.RBI,
			ops:      r.stats.OPS,
			pa:       r.stats.PlateAppearances,
		})
	}
	sortHotBatsRows(rows)
	if len(rows) > 3 {
		rows = rows[:3]
	}
	return hotBatsTeamPayload{rows: rows, loaded: true}
}

// sortHotBatsRows ranks in place by OPS desc, then HR desc, then AVG desc.
// Lifted out of computeHotBatsTeam so the comparator can be unit-tested
// without spinning up the API client.
func sortHotBatsRows(rows []hotBatsRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		oi, oj := parseStatFloat(rows[i].ops), parseStatFloat(rows[j].ops)
		if oi != oj {
			return oi > oj
		}
		if rows[i].homeRuns != rows[j].homeRuns {
			return rows[i].homeRuns > rows[j].homeRuns
		}
		ai, aj := parseStatFloat(rows[i].avg), parseStatFloat(rows[j].avg)
		return ai > aj
	})
}

// dispatchBullpen starts the bullpen fetch for the current game.
// Returns nil when the result is already cached or in-flight.
func (m *Model) dispatchBullpen() tea.Cmd {
	fetchCmd := m.fetchBullpenIfNeeded()
	if fetchCmd == nil {
		return nil
	}
	spinnerCmd := m.startPregameSpinner()
	return tea.Batch(spinnerCmd, fetchCmd)
}

// fetchBullpenIfNeeded reserves the bullpen cache slot and returns a
// tea.Cmd that fans out the roster + appearance fetches in parallel.
func (m Model) fetchBullpenIfNeeded() tea.Cmd {
	game := m.currentGame
	if game == nil || game.GameData == nil {
		return nil
	}
	data := m.ensurePregameData(game.ID)
	if data.bullpen != nil {
		return nil
	}
	data.bullpen = &bullpenPayload{}

	gameID := game.ID
	awayTeamID := game.GameData.Teams.Away.ID
	homeTeamID := game.GameData.Teams.Home.ID
	awayStarterID := game.GameData.ProbablePitchers.Away.ID
	homeStarterID := game.GameData.ProbablePitchers.Home.ID
	season := time.Now().Year()
	gameDate := game.GameData.Datetime.DateTime
	if gameDate.IsZero() {
		gameDate = game.GameDate
	}
	if gameDate.IsZero() {
		gameDate = time.Now()
	}
	client := m.apiClient

	return func() tea.Msg {
		var away, home bullpenTeamPayload
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			away = computeBullpenTeam(client, awayTeamID, awayStarterID, season, gameDate)
		}()
		go func() {
			defer wg.Done()
			home = computeBullpenTeam(client, homeTeamID, homeStarterID, season, gameDate)
		}()
		wg.Wait()
		return pregameDataLoadedMsg{
			gameID:  gameID,
			tab:     PregameTabBullpen,
			payload: bullpenPayload{away: &away, home: &home},
		}
	}
}

// computeBullpenTeam fetches the team's pitchers, drops the probable
// starter, fans out per-pitcher appearance fetches, and computes the
// status for each reliever. Errors mid-flight are reported on the
// returned payload rather than dropping the team.
func computeBullpenTeam(client *api.Client, teamID, starterID, season int, gameDate time.Time) bullpenTeamPayload {
	pitchers, err := client.FetchTeamPitchers(teamID)
	if err != nil {
		return bullpenTeamPayload{err: err, loaded: true}
	}
	relievers := make([]api.RosterPlayer, 0, len(pitchers))
	for _, p := range pitchers {
		if p.ID == starterID {
			continue
		}
		relievers = append(relievers, p)
	}
	if len(relievers) == 0 {
		return bullpenTeamPayload{loaded: true}
	}

	type result struct {
		pitcher     api.RosterPlayer
		appearances []api.PitcherAppearance
	}
	results := make([]result, len(relievers))
	var wg sync.WaitGroup
	for i := range relievers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			apps, ferr := client.FetchPitcherAppearances(relievers[i].ID, season, bullpenRecentAppearances)
			if ferr != nil {
				return
			}
			results[i] = result{pitcher: relievers[i], appearances: apps}
		}(i)
	}
	wg.Wait()

	rows := make([]bullpenReliever, 0, len(results))
	for _, r := range results {
		var appearances []bullpenAppearance
		for _, a := range r.appearances {
			appearances = append(appearances, bullpenAppearance{
				date:           a.Date,
				inningsPitched: a.InningsPitched,
				pitches:        a.Pitches,
			})
		}
		rows = append(rows, bullpenReliever{
			name:        r.pitcher.FullName,
			status:      computeBullpenStatus(appearances, gameDate),
			appearances: appearances,
		})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		// Sort by status (Unavail first, then Limited, then Ready), name
		// as tiebreaker. Puts the high-friction relievers up top.
		if rows[i].status != rows[j].status {
			return rows[i].status > rows[j].status
		}
		return rows[i].name < rows[j].name
	})
	return bullpenTeamPayload{relievers: rows, loaded: true}
}

// computeBullpenStatus applies the documented heuristic to a reliever's
// most recent appearance. See bullpen-* constants for thresholds.
func computeBullpenStatus(appearances []bullpenAppearance, gameDate time.Time) bullpenStatusKind {
	if len(appearances) == 0 {
		return bullpenReady
	}
	last := appearances[0]
	daysAgo := daysBetween(last.date, gameDate)
	if daysAgo < 0 {
		// Date parse failure: treat as ready rather than fabricate a status.
		return bullpenReady
	}
	if daysAgo <= bullpenRecentDaysWindow {
		if last.pitches >= bullpenUnavailPitchThreshold {
			return bullpenUnavail
		}
		return bullpenLimited
	}
	return bullpenReady
}

// daysBetween returns the absolute number of days between the YYYY-MM-DD
// `pastDate` and `gameDate`. Returns -1 on parse failure.
func daysBetween(pastDate string, gameDate time.Time) int {
	t, err := time.Parse("2006-01-02", pastDate)
	if err != nil {
		return -1
	}
	gameDay := time.Date(gameDate.Year(), gameDate.Month(), gameDate.Day(), 0, 0, 0, 0, time.UTC)
	pastDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	hours := gameDay.Sub(pastDay).Hours()
	if hours < 0 {
		hours = -hours
	}
	return int(hours / 24)
}

// dispatchH2H starts the head-to-head fetch for the current game.
// Returns nil when the result is already cached or in-flight.
func (m *Model) dispatchH2H() tea.Cmd {
	fetchCmd := m.fetchH2HIfNeeded()
	if fetchCmd == nil {
		return nil
	}
	spinnerCmd := m.startPregameSpinner()
	return tea.Batch(spinnerCmd, fetchCmd)
}

// fetchH2HIfNeeded reserves the h2h slot and returns a tea.Cmd that
// fetches the season schedule for the matchup and aggregates it.
func (m Model) fetchH2HIfNeeded() tea.Cmd {
	game := m.currentGame
	if game == nil || game.GameData == nil {
		return nil
	}
	data := m.ensurePregameData(game.ID)
	if data.h2h != nil {
		return nil
	}
	data.h2h = &h2hPayload{}

	gameID := game.ID
	awayID := game.GameData.Teams.Away.ID
	homeID := game.GameData.Teams.Home.ID
	season := time.Now().Year()
	client := m.apiClient

	return func() tea.Msg {
		var payload h2hPayload
		meetings, err := client.FetchHeadToHeadSchedule(awayID, homeID, season)
		if err != nil {
			payload.err = err
			payload.loaded = true
		} else {
			payload = aggregateH2H(meetings, awayID, homeID)
		}
		return pregameDataLoadedMsg{
			gameID:  gameID,
			tab:     PregameTabH2H,
			payload: payload,
		}
	}
}

// aggregateH2H walks the meetings list and returns the rolled-up
// totals, oriented to the current game's away/home assignment.
// The meetings list is expected to contain Final games only.
func aggregateH2H(meetings []api.Game, currentAwayID, currentHomeID int) h2hPayload {
	p := h2hPayload{loaded: true}
	if len(meetings) == 0 {
		return p
	}
	// Track the most recent meeting by date.
	var lastDate string
	for _, g := range meetings {
		// Map the past game's sides to the current game's away/home.
		var pastAwayRuns, pastHomeRuns int
		var pastAwayID, pastHomeID int
		pastAwayID = g.Teams.Away.Team.ID
		pastHomeID = g.Teams.Home.Team.ID
		pastAwayRuns = g.Teams.Away.Score
		pastHomeRuns = g.Teams.Home.Score

		var awayRuns, homeRuns int
		switch {
		case pastAwayID == currentAwayID && pastHomeID == currentHomeID:
			awayRuns, homeRuns = pastAwayRuns, pastHomeRuns
		case pastAwayID == currentHomeID && pastHomeID == currentAwayID:
			awayRuns, homeRuns = pastHomeRuns, pastAwayRuns
		default:
			// Defensive: skip games that don't match the matchup.
			continue
		}

		p.games++
		p.awayRunsTotal += awayRuns
		p.homeRunsTotal += homeRuns
		switch {
		case awayRuns > homeRuns:
			p.awayWins++
		case homeRuns > awayRuns:
			p.homeWins++
		}
		margin := awayRuns - homeRuns
		if margin < 0 {
			margin = -margin
		}
		if margin == 1 {
			p.oneRunGames++
		}
		if margin > p.largestMargin {
			p.largestMargin = margin
		}

		date := g.GameDate.Format("2006-01-02")
		if date > lastDate {
			lastDate = date
			p.lastMeeting = &h2hMeeting{
				date:     date,
				awayRuns: awayRuns,
				homeRuns: homeRuns,
			}
		}
	}
	return p
}

// parseStatFloat parses an MLB-style stat string (".567", "1.506") into
// a float64. Returns 0 on parse failure; the leading dot is tolerated
// because strconv.ParseFloat accepts it.
func parseStatFloat(s string) float64 {
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

// fetchPitcherDetailIfNeeded kicks off arsenal + recent-starts fetches
// for both probable pitchers, but only when no prior fetch has been
// recorded for this game. Returns nil when the cache is already
// populated or in-flight.
func (m Model) fetchPitcherDetailIfNeeded() tea.Cmd {
	game := m.currentGame
	if game == nil || game.GameData == nil {
		return nil
	}
	if m.pregameData == nil {
		m.pregameData = map[int]*pregameGameData{}
	}
	data, ok := m.pregameData[game.ID]
	if !ok {
		data = &pregameGameData{}
		m.pregameData[game.ID] = data
	}
	if data.pitcherDetail != nil {
		// Either loaded or in-flight; nothing to do.
		return nil
	}
	// Reserve the slot to prevent duplicate dispatches.
	data.pitcherDetail = &pitcherDetailPayload{}

	gameID := game.ID
	season := time.Now().Year()
	awayID := game.GameData.ProbablePitchers.Away.ID
	homeID := game.GameData.ProbablePitchers.Home.ID
	client := m.apiClient

	return func() tea.Msg {
		payload := pitcherDetailPayload{loaded: true}
		if awayID != 0 {
			payload.awayArsenal, payload.awayErr = client.FetchPitcherArsenal(awayID, season)
			if payload.awayErr == nil {
				payload.awayLog, payload.awayErr = client.FetchPitcherGameLog(awayID, season, 5)
			}
		}
		if homeID != 0 {
			payload.homeArsenal, payload.homeErr = client.FetchPitcherArsenal(homeID, season)
			if payload.homeErr == nil {
				payload.homeLog, payload.homeErr = client.FetchPitcherGameLog(homeID, season, 5)
			}
		}
		return pregameDataLoadedMsg{
			gameID:  gameID,
			tab:     PregameTabPitchers,
			payload: payload,
		}
	}
}
