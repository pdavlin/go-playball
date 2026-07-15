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
	Weather       api.Weather // zero value when no forecast is posted yet
	DayNight      string      // "day" / "night", "" when unknown
	Home          TeamCtx
	Away          TeamCtx
	Probables     [2]ProbableCtx // [0] = away, [1] = home
	Lineups       [2]LineupCtx   // [0] = away, [1] = home; zero value = "not posted"
}

// TeamCtx is one side's structured context. The form fields are empty
// when the standings fetch failed or the team isn't in a division race
// (spring training, WBC).
type TeamCtx struct {
	Name         string
	Abbreviation string
	Record       string
	Streak       string // e.g. "W3"
	LastTen      string // e.g. "7-3"
	DivisionRank string // e.g. "2"
	GamesBack    string // e.g. "1.5" or "-"
	SeasonHit    *api.HittingLine
}

// ProbableCtx is one side's probable starter context.
type ProbableCtx struct {
	Name         string
	HandsThrows  string // "RHP" / "LHP", "" when unknown
	SeasonLine   *api.PitchingLine
	Arsenal      []api.ArsenalPitch
	RecentStarts []api.GameLogStart
}

const (
	// recentStartsLimit is how many of the starter's latest outings go
	// into the prompt.
	recentStartsLimit = 3
	// recentGamesWindow is the rolling-game window for lineup batters'
	// recent form.
	recentGamesWindow = 7
)

// BuildContext gathers everything the prompt needs for one game. It fans
// out the four extra MLB stat fetches in parallel; any individual failure
// degrades to a nil pointer rather than aborting.
func BuildContext(ctx context.Context, c *api.Client, g *api.Game) (Context, error) {
	if g == nil {
		return Context{}, fmt.Errorf("scouting: nil game")
	}

	// Snapshot the schedule-side team identity before hydration; the live
	// feed leaves top-level teams.* empty so we'd otherwise lose Name and
	// Abbreviation.
	awaySnap := api.SnapshotAwayIdentity(g)
	homeSnap := api.SnapshotHomeIdentity(g)

	// Schedule-view games carry only team + linescore. Hydrate from the
	// live feed so ProbablePitchers, battingOrder, and gameData.players are
	// available. Failure is non-fatal — we fall back to whatever the
	// caller-supplied game has.
	if (g.GameData == nil || g.LiveData == nil) && g.ID != 0 && c != nil {
		if full, err := c.FetchGame(g.ID); err == nil && full != nil {
			g = full
		}
	}

	awayID := api.ResolveTeam(g, "away", awaySnap)
	homeID := api.ResolveTeam(g, "home", homeSnap)

	out := Context{
		GamePk:        g.ID,
		GameDateLocal: formatGameDate(g),
		Venue:         venueName(g),
		Away: TeamCtx{
			Name:         awayID.Name,
			Abbreviation: awayID.Abbreviation,
			Record:       formatRecord(awayID.LeagueRecord),
		},
		Home: TeamCtx{
			Name:         homeID.Name,
			Abbreviation: homeID.Abbreviation,
			Record:       formatRecord(homeID.LeagueRecord),
		},
	}

	if g.GameData != nil {
		out.Weather = g.GameData.Weather
		out.DayNight = g.GameData.Datetime.DayNight
	}

	awayTeamID := awayID.ID
	homeTeamID := homeID.ID
	season := seasonYear(g)

	var (
		awayPitcher, homePitcher *api.PitchingLine
		awayArsenal, homeArsenal []api.ArsenalPitch
		awayStarts, homeStarts   []api.GameLogStart
		awayHit, homeHit         *api.HittingLine
		awayForm, homeForm       teamForm
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
			awayPitcher, _ = c.FetchPitcherSeasonStats(awayP.ID, season)
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			awayArsenal, _ = c.FetchPitcherArsenal(awayP.ID, season)
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			awayStarts, _ = c.FetchPitcherGameLog(awayP.ID, season, recentStartsLimit)
		}()
	}
	if homeP.ID != 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			homePitcher, _ = c.FetchPitcherSeasonStats(homeP.ID, season)
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			homeArsenal, _ = c.FetchPitcherArsenal(homeP.ID, season)
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			homeStarts, _ = c.FetchPitcherGameLog(homeP.ID, season, recentStartsLimit)
		}()
	}
	if awayTeamID != 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			awayHit, _ = c.FetchTeamHittingStats(awayTeamID, season)
		}()
	}
	if homeTeamID != 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			homeHit, _ = c.FetchTeamHittingStats(homeTeamID, season)
		}()
	}
	if awayTeamID != 0 || homeTeamID != 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			recs, err := c.FetchStandings()
			if err != nil {
				return
			}
			awayForm = teamFormFromStandings(recs, awayTeamID)
			homeForm = teamFormFromStandings(recs, homeTeamID)
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
	awayForm.apply(&out.Away)
	homeForm.apply(&out.Home)
	out.Probables[0] = ProbableCtx{
		Name:         awayP.FullName,
		HandsThrows:  pitcherHand(g, awayP.ID),
		SeasonLine:   awayPitcher,
		Arsenal:      awayArsenal,
		RecentStarts: awayStarts,
	}
	out.Probables[1] = ProbableCtx{
		Name:         homeP.FullName,
		HandsThrows:  pitcherHand(g, homeP.ID),
		SeasonLine:   homePitcher,
		Arsenal:      homeArsenal,
		RecentStarts: homeStarts,
	}
	out.Lineups[0] = awayLineup
	out.Lineups[1] = homeLineup

	return out, nil
}

// teamForm is the standings-derived slice of TeamCtx, gathered in one
// fetch for both sides.
type teamForm struct {
	streak, lastTen, divisionRank, gamesBack string
}

func (f teamForm) apply(t *TeamCtx) {
	t.Streak = f.streak
	t.LastTen = f.lastTen
	t.DivisionRank = f.divisionRank
	t.GamesBack = f.gamesBack
}

func teamFormFromStandings(recs []api.DivisionStandings, teamID int) teamForm {
	if teamID == 0 {
		return teamForm{}
	}
	for _, div := range recs {
		for _, tr := range div.TeamRecords {
			if tr.Team.ID != teamID {
				continue
			}
			f := teamForm{
				streak:       tr.Streak.StreakCode,
				divisionRank: tr.DivisionRank,
				gamesBack:    tr.GamesBack,
			}
			for _, sr := range tr.LastTenGames.SplitRecords {
				if sr.Type == "lastTen" {
					f.lastTen = fmt.Sprintf("%d-%d", sr.Wins, sr.Losses)
					break
				}
			}
			return f
		}
	}
	return teamForm{}
}

// pitcherHand maps the live-feed pitchHand code for playerID to the
// conventional RHP/LHP label, or "" when unknown.
func pitcherHand(g *api.Game, playerID int) string {
	if g.GameData == nil || playerID == 0 {
		return ""
	}
	p, ok := g.GameData.Players[fmt.Sprintf("ID%d", playerID)]
	if !ok {
		return ""
	}
	switch p.PitchHand.Code {
	case "R":
		return "RHP"
	case "L":
		return "LHP"
	}
	return ""
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
		wg.Add(1)
		go func() {
			defer wg.Done()
			recent, _ := c.FetchHitterLastXGames(batters[i].PlayerID, season, recentGamesWindow)
			batters[i].Recent = recent
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
