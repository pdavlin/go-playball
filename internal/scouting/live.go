package scouting

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pdavlin/go-playball/internal/api"
)

// GameState discriminates the tense of a scouting report. The zero value is
// the pregame preview, so every existing caller and test that builds a bare
// Context keeps the original behavior without change.
type GameState int

const (
	// StatePregame is the preview tense for a game that has not started.
	StatePregame GameState = iota
	// StateInProgress is the tense for a game already under way. It swaps in
	// the in-progress system prompt and appends the live-state fact block.
	StateInProgress
)

// LiveCtx carries the in-game facts an in-progress report reasons from. It is
// sourced entirely from the live feed the app already fetches (linescore and
// boxscore); nothing here is a new network call. Any fact the live model does
// not supply is left at its zero value and the renderer omits it.
type LiveCtx struct {
	Inning        int    // current inning number, e.g. 6
	InningState   string // "Top" / "Middle" / "Bottom" / "End", "" when unknown
	InningOrdinal string // "6th", "" when the feed has no ordinal yet
	Outs          int    // outs in the current half-inning
	AwayRuns      int    // away team's runs so far
	HomeRuns      int    // home team's runs so far
	OnFirst       bool   // a runner occupies first base
	OnSecond      bool   // a runner occupies second base
	OnThird       bool   // a runner occupies third base
	// Starters holds each side's starting pitcher's line so far, indexed
	// [0] = away, [1] = home. Present is false when the boxscore has no
	// starter line yet.
	Starters [2]LiveStarterLine
	// Standouts holds each side's best batting lines so far, indexed
	// [0] = away, [1] = home. Empty when no batter has a hit yet.
	Standouts [2][]LiveHitterLine
}

// LiveStarterLine is one starter's boxscore pitching line up to the current
// moment. Pitches is 0 when the feed omits the pitch count.
type LiveStarterLine struct {
	Name    string
	IP      string // innings pitched so far, e.g. "5.1"
	H       int
	R       int
	ER      int
	K       int
	Pitches int
	Present bool
}

// LiveHitterLine is one batter's game line so far. Only batters with at least
// one hit are considered a standout.
type LiveHitterLine struct {
	Name string
	AB   int
	H    int
	HR   int
	RBI  int
}

// liveStandoutsPerSide caps how many batting lines per team go into the live
// block, keeping the prompt tight and steering the model toward the bats that
// have actually done something.
const liveStandoutsPerSide = 2

// isInProgress reports whether g is in the Live abstract game state, checking
// the hydrated gameData first and falling back to the schedule-side status.
func isInProgress(g *api.Game) bool {
	return abstractState(g) == "Live"
}

func abstractState(g *api.Game) string {
	if g == nil {
		return ""
	}
	if g.GameData != nil && g.GameData.Status.AbstractGameState != "" {
		return g.GameData.Status.AbstractGameState
	}
	return g.Status.AbstractGameState
}

// buildLiveContext assembles the in-game fact block from the live feed. It
// returns nil when no live data is present (e.g. hydration failed), which the
// renderer treats as "no live specifics" while still using the in-progress
// tense. It invents nothing: every field maps to a value already in the feed.
func buildLiveContext(g *api.Game) *LiveCtx {
	if g == nil || g.LiveData == nil {
		return nil
	}
	ls := g.LiveData.Linescore
	live := &LiveCtx{
		Inning:        ls.CurrentInning,
		InningState:   ls.InningState,
		InningOrdinal: ls.CurrentInningOrdinal,
		Outs:          ls.Outs,
		AwayRuns:      ls.Teams.Away.Runs,
		HomeRuns:      ls.Teams.Home.Runs,
		OnFirst:       ls.Offense.First != nil,
		OnSecond:      ls.Offense.Second != nil,
		OnThird:       ls.Offense.Third != nil,
	}
	bs := g.LiveData.Boxscore
	live.Starters[0] = starterLine(bs.Teams.Away)
	live.Starters[1] = starterLine(bs.Teams.Home)
	live.Standouts[0] = liveStandouts(bs.Teams.Away)
	live.Standouts[1] = liveStandouts(bs.Teams.Home)
	return live
}

// starterLine reads the team's starting pitcher's in-game line. The boxscore
// Pitchers slice is ordered by appearance, so the first id is the starter even
// after he has been pulled — his line stays in the boxscore.
func starterLine(team api.BoxscoreTeam) LiveStarterLine {
	if len(team.Pitchers) == 0 {
		return LiveStarterLine{}
	}
	p, ok := team.Players[fmt.Sprintf("ID%d", team.Pitchers[0])]
	if !ok || p.Stats.Pitching == nil {
		return LiveStarterLine{}
	}
	ps := p.Stats.Pitching
	return LiveStarterLine{
		Name:    p.Person.FullName,
		IP:      ps.InningsPitched,
		H:       ps.Hits,
		R:       ps.Runs,
		ER:      ps.EarnedRuns,
		K:       ps.StrikeOuts,
		Pitches: ps.PitchesThrown,
		Present: true,
	}
}

// liveStandouts ranks a team's batters by game impact (hits weighted, then
// homers and RBI) and returns the top few who have at least one hit. Mirrors
// the recap package's topHitters heuristic but scoped to in-game batting.
func liveStandouts(team api.BoxscoreTeam) []LiveHitterLine {
	type ranked struct {
		line  LiveHitterLine
		score int
	}
	var pool []ranked
	for _, p := range team.Players {
		if p.Stats.Batting == nil || p.Stats.Batting.Hits == 0 {
			continue
		}
		b := p.Stats.Batting
		pool = append(pool, ranked{
			line: LiveHitterLine{
				Name: p.Person.FullName,
				AB:   b.AtBats,
				H:    b.Hits,
				HR:   b.HomeRuns,
				RBI:  b.RBI,
			},
			score: b.Hits*10 + b.HomeRuns*5 + b.RBI,
		})
	}
	sort.SliceStable(pool, func(i, j int) bool {
		if pool[i].score != pool[j].score {
			return pool[i].score > pool[j].score
		}
		return pool[i].line.Name < pool[j].line.Name
	})
	n := liveStandoutsPerSide
	if n > len(pool) {
		n = len(pool)
	}
	out := make([]LiveHitterLine, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, pool[i].line)
	}
	return out
}

// writeLiveBlock appends the "Live game state" fact block to the user prompt.
// It is a no-op when live is nil so an in-progress game whose feed failed to
// hydrate still renders (just without live specifics).
func writeLiveBlock(b *strings.Builder, live *LiveCtx) {
	if live == nil {
		return
	}
	fmt.Fprintln(b)
	fmt.Fprintln(b, "Live game state (the app shows these to the fan — context only, do not restate):")
	fmt.Fprintf(b, "  Situation: %s, %d out\n", inningPhrase(live), live.Outs)
	if live.AwayRuns == 0 && live.HomeRuns == 0 {
		fmt.Fprintln(b, "  Score: scoreless, no runs yet")
	} else {
		fmt.Fprintf(b, "  Score: Away %d, Home %d\n", live.AwayRuns, live.HomeRuns)
	}
	fmt.Fprintf(b, "  Bases: %s\n", basesPhrase(live))
	writeLiveStarter(b, "Away", live.Starters[0])
	writeLiveStarter(b, "Home", live.Starters[1])
	writeLiveStandouts(b, "Away", live.Standouts[0])
	writeLiveStandouts(b, "Home", live.Standouts[1])
}

func writeLiveStarter(b *strings.Builder, side string, s LiveStarterLine) {
	if !s.Present || s.Name == "" {
		return
	}
	line := fmt.Sprintf("  %s starter %s so far: %s IP, %d H, %d R, %d ER, %d K",
		side, s.Name, s.IP, s.H, s.R, s.ER, s.K)
	if s.Pitches > 0 {
		line += fmt.Sprintf(", %d pitches", s.Pitches)
	}
	fmt.Fprintln(b, line)
}

func writeLiveStandouts(b *strings.Builder, side string, hitters []LiveHitterLine) {
	if len(hitters) == 0 {
		return
	}
	fmt.Fprintf(b, "  %s standouts so far:\n", side)
	for _, h := range hitters {
		line := fmt.Sprintf("    %s: %d-for-%d", h.Name, h.H, h.AB)
		if h.HR > 0 {
			line += fmt.Sprintf(", %d HR", h.HR)
		}
		if h.RBI > 0 {
			line += fmt.Sprintf(", %d RBI", h.RBI)
		}
		fmt.Fprintln(b, line)
	}
}

// inningPhrase renders the half-inning as prose grounding, e.g. "bottom of
// the 6th". Falls back to the bare inning number when no ordinal is supplied.
func inningPhrase(l *LiveCtx) string {
	ord := l.InningOrdinal
	if ord == "" {
		ord = fmt.Sprintf("%d", l.Inning)
	}
	state := strings.ToLower(l.InningState)
	if state == "" {
		return "the " + ord
	}
	return state + " of the " + ord
}

// basesPhrase renders the base-out occupancy as prose grounding.
func basesPhrase(l *LiveCtx) string {
	var on []string
	if l.OnFirst {
		on = append(on, "first")
	}
	if l.OnSecond {
		on = append(on, "second")
	}
	if l.OnThird {
		on = append(on, "third")
	}
	switch len(on) {
	case 0:
		return "empty"
	case 1:
		return "runner on " + on[0]
	default:
		return "runners on " + strings.Join(on[:len(on)-1], ", ") + " and " + on[len(on)-1]
	}
}
