// Package recap builds LLM postgame recap reports for completed MLB
// games. Sibling to internal/scouting; shares the LLM provider layer and
// the on-disk cache (via internal/reportcache) but ships its own context
// shape and prompt.
package recap

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/pdavlin/go-playball/internal/api"
)

// topPerformers is how many hitters and pitchers per side make the
// standouts block. The LLM picks one or two from this short list per
// section; pre-filtering keeps the prompt small and stops the model
// from latching onto a 1-for-4 line because the name is recognizable.
const topPerformers = 3

// ErrNotFinal is returned by BuildContext when the game is not in a
// Final abstract state. Modal surfaces it as "recap available only for
// final games."
var ErrNotFinal = errors.New("recap: game is not final")

// ErrIncompletePayload is returned when a Final game's live feed lacks
// the minimum data needed for a recap (no decisions and no scoring).
// In practice this only happens for malformed or mid-update payloads.
var ErrIncompletePayload = errors.New("recap: live feed missing decisions and scoring")

// Context is the structured input to RenderPrompt.
type Context struct {
	GamePk        int
	GameDateLocal string
	Venue         string
	Away          TeamScore
	Home          TeamScore
	Decisions     DecisionsCtx
	Linescore     []InningPair
	Scoring       []ScoringPlay
	Standouts     [2]TeamStandouts // [0] = away, [1] = home
}

type TeamScore struct {
	Name         string // e.g. "Cincinnati Reds"
	Nickname     string // e.g. "Reds" — MLB API teamName field, authoritative
	Abbreviation string // e.g. "CIN"
	Runs         int
	Hits         int
	Errors       int
}

type DecisionsCtx struct {
	Winner *PitcherDecision
	Loser  *PitcherDecision
	Save   *PitcherDecision
}

type PitcherDecision struct {
	Name string
}

type InningPair struct {
	Inning     int
	Away       int
	Home       int
	AwayPlayed bool
	HomePlayed bool
}

type ScoringPlay struct {
	Inning      int
	HalfInning  string // "top" / "bottom"
	Description string
	AwayScore   int
	HomeScore   int
}

type TeamStandouts struct {
	Hitters  []HitterLine
	Pitchers []PitcherLine
}

type HitterLine struct {
	Name string
	AB   int
	H    int
	HR   int
	RBI  int
}

type PitcherLine struct {
	Name     string
	IP       string
	K        int
	BB       int
	ER       int
	Decision string // "W", "L", "SV", or ""
}

// BuildContext gathers everything the recap prompt needs. Hydrates the
// game via the live feed when GameData/LiveData are missing. Returns
// ErrNotFinal or ErrIncompletePayload when the game cannot be recapped.
func BuildContext(ctx context.Context, c *api.Client, g *api.Game) (Context, error) {
	if g == nil {
		return Context{}, fmt.Errorf("recap: nil game")
	}

	// Snapshot schedule-side team identity before hydration overwrites g
	// (the live feed leaves top-level teams.* empty).
	awaySnap := api.SnapshotAwayIdentity(g)
	homeSnap := api.SnapshotHomeIdentity(g)

	if (g.GameData == nil || g.LiveData == nil) && g.ID != 0 && c != nil {
		full, err := c.FetchGame(g.ID)
		if err != nil {
			return Context{}, fmt.Errorf("recap: failed to load game %d: %w", g.ID, err)
		}
		g = full
	}

	if g.LiveData == nil {
		return Context{}, ErrIncompletePayload
	}

	if !isFinal(g) {
		return Context{}, ErrNotFinal
	}

	awayID := api.ResolveTeam(g, "away", awaySnap)
	homeID := api.ResolveTeam(g, "home", homeSnap)

	ls := g.LiveData.Linescore
	out := Context{
		GamePk:        g.ID,
		GameDateLocal: formatGameDate(g),
		Venue:         venueName(g),
		Away: TeamScore{
			Name:         awayID.Name,
			Nickname:     awayID.TeamName,
			Abbreviation: awayID.Abbreviation,
			Runs:         ls.Teams.Away.Runs,
			Hits:         ls.Teams.Away.Hits,
			Errors:       ls.Teams.Away.Errors,
		},
		Home: TeamScore{
			Name:         homeID.Name,
			Nickname:     homeID.TeamName,
			Abbreviation: homeID.Abbreviation,
			Runs:         ls.Teams.Home.Runs,
			Hits:         ls.Teams.Home.Hits,
			Errors:       ls.Teams.Home.Errors,
		},
	}

	out.Decisions = extractDecisions(g.LiveData.Decisions)
	out.Linescore = buildLinescore(ls.Innings)
	out.Scoring = collectScoringPlays(g.LiveData.Plays.AllPlays)
	out.Standouts[0] = buildStandouts(g.LiveData.Boxscore.Teams.Away, g.LiveData.Decisions)
	out.Standouts[1] = buildStandouts(g.LiveData.Boxscore.Teams.Home, g.LiveData.Decisions)

	if !hasMinimalData(out) {
		return Context{}, ErrIncompletePayload
	}

	return out, nil
}

func isFinal(g *api.Game) bool {
	if g.GameData != nil && g.GameData.Status.AbstractGameState == "Final" {
		return true
	}
	return g.Status.AbstractGameState == "Final"
}

func hasMinimalData(c Context) bool {
	if c.Decisions.Winner != nil || c.Decisions.Loser != nil || c.Decisions.Save != nil {
		return true
	}
	if c.Away.Runs > 0 || c.Home.Runs > 0 {
		return true
	}
	return false
}

func extractDecisions(d api.Decisions) DecisionsCtx {
	out := DecisionsCtx{}
	if d.Winner != nil {
		out.Winner = &PitcherDecision{Name: d.Winner.FullName}
	}
	if d.Loser != nil {
		out.Loser = &PitcherDecision{Name: d.Loser.FullName}
	}
	if d.Save != nil {
		out.Save = &PitcherDecision{Name: d.Save.FullName}
	}
	return out
}

func buildLinescore(innings []api.Inning) []InningPair {
	out := make([]InningPair, 0, len(innings))
	for _, in := range innings {
		out = append(out, InningPair{
			Inning:     in.Num,
			Away:       in.Away.RunsVal(),
			Home:       in.Home.RunsVal(),
			AwayPlayed: in.Away.WasPlayed(),
			HomePlayed: in.Home.WasPlayed(),
		})
	}
	return out
}

func collectScoringPlays(plays []api.Play) []ScoringPlay {
	var out []ScoringPlay
	for _, p := range plays {
		if !p.About.IsScoringPlay {
			continue
		}
		out = append(out, ScoringPlay{
			Inning:      p.About.Inning,
			HalfInning:  p.About.HalfInning,
			Description: p.Result.Description,
			AwayScore:   p.Result.AwayScore,
			HomeScore:   p.Result.HomeScore,
		})
	}
	return out
}

func buildStandouts(team api.BoxscoreTeam, decisions api.Decisions) TeamStandouts {
	hitters := topHitters(team.Players)
	pitchers := topPitchers(team.Players, decisions)
	return TeamStandouts{Hitters: hitters, Pitchers: pitchers}
}

func topHitters(players map[string]api.BoxscorePlayer) []HitterLine {
	type ranked struct {
		line  HitterLine
		score int
	}
	var pool []ranked
	for _, p := range players {
		if p.Stats.Batting == nil || p.Stats.Batting.AtBats == 0 {
			continue
		}
		b := p.Stats.Batting
		pool = append(pool, ranked{
			line: HitterLine{
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
	n := topPerformers
	if n > len(pool) {
		n = len(pool)
	}
	out := make([]HitterLine, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, pool[i].line)
	}
	return out
}

func topPitchers(players map[string]api.BoxscorePlayer, decisions api.Decisions) []PitcherLine {
	type ranked struct {
		line  PitcherLine
		score int
		outs  int
	}
	var pool []ranked
	for _, p := range players {
		if p.Stats.Pitching == nil {
			continue
		}
		ps := p.Stats.Pitching
		line := PitcherLine{
			Name: p.Person.FullName,
			IP:   ps.InningsPitched,
			K:    ps.StrikeOuts,
			BB:   ps.BaseOnBalls,
			ER:   ps.EarnedRuns,
		}
		line.Decision = decisionFor(p.Person.ID, decisions)
		pool = append(pool, ranked{
			line:  line,
			score: ps.StrikeOuts - 2*ps.EarnedRuns,
			outs:  parseInningsToOuts(ps.InningsPitched),
		})
	}
	sort.SliceStable(pool, func(i, j int) bool {
		if pool[i].score != pool[j].score {
			return pool[i].score > pool[j].score
		}
		if pool[i].outs != pool[j].outs {
			return pool[i].outs > pool[j].outs
		}
		return pool[i].line.Name < pool[j].line.Name
	})
	n := topPerformers
	if n > len(pool) {
		n = len(pool)
	}
	out := make([]PitcherLine, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, pool[i].line)
	}
	return out
}

func decisionFor(playerID int, d api.Decisions) string {
	if d.Winner != nil && d.Winner.ID == playerID {
		return "W"
	}
	if d.Loser != nil && d.Loser.ID == playerID {
		return "L"
	}
	if d.Save != nil && d.Save.ID == playerID {
		return "SV"
	}
	return ""
}

// parseInningsToOuts converts "7.1" → 22 outs, "7.0" → 21, "7" → 21,
// "0.2" → 2. Anything malformed returns 0. Only used as a sort tiebreak.
func parseInningsToOuts(ip string) int {
	whole := 0
	frac := 0
	dot := -1
	for i := 0; i < len(ip); i++ {
		if ip[i] == '.' {
			dot = i
			break
		}
	}
	if dot < 0 {
		fmt.Sscanf(ip, "%d", &whole)
	} else {
		fmt.Sscanf(ip[:dot], "%d", &whole)
		if dot+1 < len(ip) {
			fmt.Sscanf(ip[dot+1:], "%d", &frac)
		}
	}
	return whole*3 + frac
}

func venueName(g *api.Game) string {
	if g.GameData == nil {
		return ""
	}
	return g.GameData.Venue.Name
}

func formatGameDate(g *api.Game) string {
	var t time.Time
	if g.GameData != nil && !g.GameData.Datetime.DateTime.IsZero() {
		t = g.GameData.Datetime.DateTime
	} else if !g.GameDate.IsZero() {
		t = g.GameDate
	} else {
		return ""
	}
	return t.Local().Format("Mon, Jan 2 at 3:04 PM MST")
}
