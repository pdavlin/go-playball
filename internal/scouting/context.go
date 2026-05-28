// Package scouting builds LLM scouting reports for upcoming MLB games. It
// is the only package in the project that imports both the api and llm
// packages; UI code only talks to scouting.
package scouting

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pdavlin/go-playball/internal/api"
)

// Context is the structured input to the prompt renderer. Optional fields
// (SeasonHit, SeasonLine) are nil when the MLB stats fetch failed or
// returned an empty splits array.
type Context struct {
	GamePk        int
	GameDateLocal string
	Venue         string
	Home          TeamCtx
	Away          TeamCtx
	Probables     [2]ProbableCtx // [0] = away, [1] = home
	Lineups       [2]LineupCtx   // [0] = away, [1] = home; zero value = "not posted"
}

// TeamCtx is one side's structured context.
type TeamCtx struct {
	Name         string
	Abbreviation string
	Record       string
	SeasonHit    *api.HittingLine
}

// ProbableCtx is one side's probable starter context.
type ProbableCtx struct {
	Name       string
	HandsThrows string
	SeasonLine *api.PitchingLine
}

// BuildContext gathers everything the prompt needs for one game. It fans
// out the four extra MLB stat fetches in parallel; any individual failure
// degrades to a nil pointer rather than aborting.
func BuildContext(ctx context.Context, c *api.Client, g *api.Game) (Context, error) {
	if g == nil {
		return Context{}, fmt.Errorf("scouting: nil game")
	}

	// Schedule-view games carry only team + linescore. Hydrate from the
	// live feed so ProbablePitchers, battingOrder, and gameData.players are
	// available. Failure is non-fatal — we fall back to whatever the
	// caller-supplied game has.
	if (g.GameData == nil || g.LiveData == nil) && g.ID != 0 && c != nil {
		if full, err := c.FetchGame(g.ID); err == nil && full != nil {
			g = full
		}
	}

	out := Context{
		GamePk:        g.ID,
		GameDateLocal: formatGameDate(g),
		Venue:         venueName(g),
		Away: TeamCtx{
			Name:         g.Teams.Away.Team.Name,
			Abbreviation: g.Teams.Away.Team.Abbreviation,
			Record:       formatRecord(g.Teams.Away.LeagueRecord),
		},
		Home: TeamCtx{
			Name:         g.Teams.Home.Team.Name,
			Abbreviation: g.Teams.Home.Team.Abbreviation,
			Record:       formatRecord(g.Teams.Home.LeagueRecord),
		},
	}

	awayID := g.Teams.Away.Team.ID
	homeID := g.Teams.Home.Team.ID
	season := seasonYear(g)

	var (
		awayPitcher, homePitcher *api.PitchingLine
		awayHit, homeHit         *api.HittingLine
		awayP, homeP             api.ProbablePitcher
	)

	if g.GameData != nil {
		awayP = g.GameData.ProbablePitchers.Away
		homeP = g.GameData.ProbablePitchers.Home
	}

	var wg sync.WaitGroup

	if awayP.ID != 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			line, _ := c.FetchPitcherSeasonStats(awayP.ID, season)
			awayPitcher = line
		}()
	}
	if homeP.ID != 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			line, _ := c.FetchPitcherSeasonStats(homeP.ID, season)
			homePitcher = line
		}()
	}
	if awayID != 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			line, _ := c.FetchTeamHittingStats(awayID, season)
			awayHit = line
		}()
	}
	if homeID != 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			line, _ := c.FetchTeamHittingStats(homeID, season)
			homeHit = line
		}()
	}

	awayLineup, homeLineup := gatherLineups(g)
	if len(awayLineup.Batters) > 0 && len(homeLineup.Batters) > 0 {
		fetchTopBatters(&wg, c, season, awayLineup.Batters)
		fetchTopBatters(&wg, c, season, homeLineup.Batters)
	}

	wg.Wait()

	out.Away.SeasonHit = awayHit
	out.Home.SeasonHit = homeHit
	out.Probables[0] = ProbableCtx{Name: awayP.FullName, SeasonLine: awayPitcher}
	out.Probables[1] = ProbableCtx{Name: homeP.FullName, SeasonLine: homePitcher}
	out.Lineups[0] = awayLineup
	out.Lineups[1] = homeLineup

	return out, nil
}

func fetchTopBatters(wg *sync.WaitGroup, c *api.Client, season int, batters []BatterCtx) {
	limit := topBatters
	if limit > len(batters) {
		limit = len(batters)
	}
	for i := 0; i < limit; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			line, _ := c.FetchBatterSeasonStats(batters[i].PlayerID, season)
			batters[i].SeasonLine = line
		}()
	}
}

func formatGameDate(g *api.Game) string {
	var t time.Time
	if g.GameData != nil && !g.GameData.Datetime.DateTime.IsZero() {
		t = g.GameData.Datetime.DateTime
	} else if !g.GameDate.IsZero() {
		t = g.GameDate
	} else {
		return "TBD"
	}
	return t.Local().Format("Mon, Jan 2 at 3:04 PM MST")
}

func venueName(g *api.Game) string {
	if g.GameData == nil {
		return ""
	}
	return g.GameData.Venue.Name
}

func formatRecord(r api.LeagueRecord) string {
	return fmt.Sprintf("%d-%d", r.Wins, r.Losses)
}

func seasonYear(g *api.Game) int {
	if g.GameData != nil && !g.GameData.Datetime.DateTime.IsZero() {
		return g.GameData.Datetime.DateTime.Year()
	}
	if !g.GameDate.IsZero() {
		return g.GameDate.Year()
	}
	return time.Now().Year()
}
