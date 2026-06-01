package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// PitchingLine is a minimal season pitching summary used by the scouting
// prompt. All fields are strings as returned by the MLB stats API.
type PitchingLine struct {
	Wins   int    `json:"wins"`
	Losses int    `json:"losses"`
	ERA    string `json:"era"`
	WHIP   string `json:"whip"`
	K9     string `json:"strikeoutsPer9Inn"`
	IP     string `json:"inningsPitched"`
}

// HittingLine is a minimal season team hitting summary used by the scouting
// prompt.
type HittingLine struct {
	AVG      string `json:"avg"`
	OBP      string `json:"obp"`
	SLG      string `json:"slg"`
	OPS      string `json:"ops"`
	HomeRuns int    `json:"homeRuns"`
	RBI      int    `json:"rbi"`
}

// statsEnvelope models the MLB stats API splits shape:
// { "stats": [{ "splits": [{ "stat": {...} }] }] }
type statsEnvelope[T any] struct {
	Stats []struct {
		Splits []struct {
			Stat T `json:"stat"`
		} `json:"splits"`
	} `json:"stats"`
}

func (c *Client) fetchStat(url string, dst any) error {
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("fetching stats: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("stats API returned status %d: %s", resp.StatusCode, string(body))
	}
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decoding stats response: %w", err)
	}
	return nil
}

// FetchPitcherSeasonStats returns a pitcher's season line, or nil,nil when
// the API returns an empty splits array (e.g., rookie pitcher with no
// rostered stats yet, or very early-season call-up).
func (c *Client) FetchPitcherSeasonStats(playerID, season int) (*PitchingLine, error) {
	url := fmt.Sprintf("%s/api/v1/people/%d/stats?stats=season&group=pitching&season=%d",
		baseURL, playerID, season)

	var env statsEnvelope[PitchingLine]
	if err := c.fetchStat(url, &env); err != nil {
		return nil, err
	}
	if len(env.Stats) == 0 || len(env.Stats[0].Splits) == 0 {
		return nil, nil
	}
	line := env.Stats[0].Splits[0].Stat
	return &line, nil
}

// FetchBatterSeasonStats returns a batter's season hitting line, or nil,nil
// when the API returns an empty splits array (e.g., a call-up with no
// rostered splits yet).
func (c *Client) FetchBatterSeasonStats(playerID, season int) (*HittingLine, error) {
	url := fmt.Sprintf("%s/api/v1/people/%d/stats?stats=season&group=hitting&season=%d",
		baseURL, playerID, season)

	var env statsEnvelope[HittingLine]
	if err := c.fetchStat(url, &env); err != nil {
		return nil, err
	}
	if len(env.Stats) == 0 || len(env.Stats[0].Splits) == 0 {
		return nil, nil
	}
	line := env.Stats[0].Splits[0].Stat
	return &line, nil
}

// FetchTeamHittingStats returns a team's season hitting line, or nil,nil
// when the API returns an empty splits array.
func (c *Client) FetchTeamHittingStats(teamID, season int) (*HittingLine, error) {
	url := fmt.Sprintf("%s/api/v1/teams/%d/stats?stats=season&group=hitting&season=%d",
		baseURL, teamID, season)

	var env statsEnvelope[HittingLine]
	if err := c.fetchStat(url, &env); err != nil {
		return nil, err
	}
	if len(env.Stats) == 0 || len(env.Stats[0].Splits) == 0 {
		return nil, nil
	}
	line := env.Stats[0].Splits[0].Stat
	return &line, nil
}

// ArsenalPitch is one entry in a pitcher's season pitch-arsenal.
type ArsenalPitch struct {
	Code        string
	Description string
	UsagePct    float64 // 0..100
	AvgVelocity float64 // mph; 0 when missing
	Count       int
}

// arsenalStat models the inner shape returned by pitchArsenal splits.
type arsenalStat struct {
	Percentage   float64 `json:"percentage"`
	Count        int     `json:"count"`
	TotalPitches int     `json:"totalPitches"`
	AverageSpeed float64 `json:"averageSpeed"`
	Type         struct {
		Code        string `json:"code"`
		Description string `json:"description"`
	} `json:"type"`
}

// FetchPitcherArsenal returns the named pitcher's season pitch arsenal,
// or nil,nil when the API returns no splits (rookies, very early season,
// or non-pitchers).
func (c *Client) FetchPitcherArsenal(playerID, season int) ([]ArsenalPitch, error) {
	url := fmt.Sprintf("%s/api/v1/people/%d/stats?stats=pitchArsenal&season=%d",
		baseURL, playerID, season)

	var env statsEnvelope[arsenalStat]
	if err := c.fetchStat(url, &env); err != nil {
		return nil, err
	}
	if len(env.Stats) == 0 || len(env.Stats[0].Splits) == 0 {
		return nil, nil
	}
	splits := env.Stats[0].Splits
	out := make([]ArsenalPitch, 0, len(splits))
	for _, s := range splits {
		out = append(out, ArsenalPitch{
			Code:        s.Stat.Type.Code,
			Description: s.Stat.Type.Description,
			UsagePct:    s.Stat.Percentage * 100,
			AvgVelocity: s.Stat.AverageSpeed,
			Count:       s.Stat.Count,
		})
	}
	return out, nil
}

// GameLogStart is one row in a pitcher's recent-starts log.
type GameLogStart struct {
	Date           string
	OpponentName   string
	InningsPitched string
	EarnedRuns     int
	Strikeouts     int
	Walks          int
	HomeRuns       int
	Pitches        int
	IsStart        bool
}

// gameLogPitchingStat models the per-split stat block in a pitching game log.
type gameLogPitchingStat struct {
	GamesStarted    int    `json:"gamesStarted"`
	InningsPitched  string `json:"inningsPitched"`
	EarnedRuns      int    `json:"earnedRuns"`
	StrikeOuts      int    `json:"strikeOuts"`
	BaseOnBalls     int    `json:"baseOnBalls"`
	HomeRuns        int    `json:"homeRuns"`
	NumberOfPitches int    `json:"numberOfPitches"`
}

// gameLogEnvelope models the gameLog response shape, which carries
// per-split fields (date, opponent) alongside the stat block.
type gameLogEnvelope struct {
	Stats []struct {
		Splits []struct {
			Date     string              `json:"date"`
			Opponent struct {
				Name string `json:"name"`
			} `json:"opponent"`
			Stat gameLogPitchingStat `json:"stat"`
		} `json:"splits"`
	} `json:"stats"`
}

// BatterWindowStats is one hitter's aggregated line over a rolling
// game window (e.g. last 7 / 15 / 30 games).
type BatterWindowStats struct {
	AVG              string
	HomeRuns         int
	RBI              int
	OPS              string
	PlateAppearances int
	GamesPlayed      int
}

// batterWindowStat models the inner shape returned by lastXGames splits.
type batterWindowStat struct {
	AVG              string `json:"avg"`
	HomeRuns         int    `json:"homeRuns"`
	RBI              int    `json:"rbi"`
	OPS              string `json:"ops"`
	PlateAppearances int    `json:"plateAppearances"`
	GamesPlayed      int    `json:"gamesPlayed"`
}

// FetchHitterLastXGames returns the hitter's aggregated stats over the
// most recent `lastX` games of the season, or nil,nil when the API
// returns no splits (didn't play in the window or non-hitter).
func (c *Client) FetchHitterLastXGames(playerID, season, lastX int) (*BatterWindowStats, error) {
	url := fmt.Sprintf("%s/api/v1/people/%d/stats?stats=lastXGames&group=hitting&season=%d&limit=%d",
		baseURL, playerID, season, lastX)

	var env statsEnvelope[batterWindowStat]
	if err := c.fetchStat(url, &env); err != nil {
		return nil, err
	}
	if len(env.Stats) == 0 || len(env.Stats[0].Splits) == 0 {
		return nil, nil
	}
	s := env.Stats[0].Splits[0].Stat
	return &BatterWindowStats{
		AVG:              s.AVG,
		HomeRuns:         s.HomeRuns,
		RBI:              s.RBI,
		OPS:              s.OPS,
		PlateAppearances: s.PlateAppearances,
		GamesPlayed:      s.GamesPlayed,
	}, nil
}

// RosterPlayer is one entry on a team's active roster.
type RosterPlayer struct {
	ID           int
	FullName     string
	Position     string // abbreviation (e.g. "1B", "C", "LF")
	PositionType string // "Pitcher", "Infielder", "Outfielder", "Catcher", "Hitter", "TwoWayPlayer"
	JerseyNumber string
}

// rosterResponse models the active-roster endpoint.
type rosterResponse struct {
	Roster []struct {
		Person struct {
			ID       int    `json:"id"`
			FullName string `json:"fullName"`
		} `json:"person"`
		JerseyNumber string `json:"jerseyNumber"`
		Position     struct {
			Abbreviation string `json:"abbreviation"`
			Type         string `json:"type"`
		} `json:"position"`
	} `json:"roster"`
}

// FetchTeamHitters returns the team's active non-pitcher roster.
// Two-way players (Ohtani-style) are included.
func (c *Client) FetchTeamHitters(teamID int) ([]RosterPlayer, error) {
	return c.fetchTeamRoster(teamID, func(positionType string) bool {
		return positionType != "Pitcher"
	})
}

// FetchTeamPitchers returns the team's active pitcher roster.
func (c *Client) FetchTeamPitchers(teamID int) ([]RosterPlayer, error) {
	return c.fetchTeamRoster(teamID, func(positionType string) bool {
		return positionType == "Pitcher" || positionType == "TwoWayPlayer"
	})
}

// fetchTeamRoster fetches the active roster and returns players whose
// position passes the include predicate.
func (c *Client) fetchTeamRoster(teamID int, include func(positionType string) bool) ([]RosterPlayer, error) {
	url := fmt.Sprintf("%s/api/v1/teams/%d/roster/active", baseURL, teamID)
	var resp rosterResponse
	if err := c.fetchStat(url, &resp); err != nil {
		return nil, err
	}
	out := make([]RosterPlayer, 0, len(resp.Roster))
	for _, r := range resp.Roster {
		if !include(r.Position.Type) {
			continue
		}
		out = append(out, RosterPlayer{
			ID:           r.Person.ID,
			FullName:     r.Person.FullName,
			Position:     r.Position.Abbreviation,
			PositionType: r.Position.Type,
			JerseyNumber: r.JerseyNumber,
		})
	}
	return out, nil
}

// PitcherAppearance is one game-log entry for a pitcher, regardless of
// whether the outing was a start or relief appearance.
type PitcherAppearance struct {
	Date           string
	OpponentName   string
	InningsPitched string
	Pitches        int
	IsStart        bool
}

// FetchPitcherAppearances returns the pitcher's most recent `limit`
// appearances (starts + relief) for the season, newest first. Returns
// nil,nil when the API returns no splits.
func (c *Client) FetchPitcherAppearances(playerID, season, limit int) ([]PitcherAppearance, error) {
	url := fmt.Sprintf("%s/api/v1/people/%d/stats?stats=gameLog&group=pitching&season=%d",
		baseURL, playerID, season)

	var env gameLogEnvelope
	if err := c.fetchStat(url, &env); err != nil {
		return nil, err
	}
	if len(env.Stats) == 0 || len(env.Stats[0].Splits) == 0 {
		return nil, nil
	}
	splits := env.Stats[0].Splits
	out := make([]PitcherAppearance, 0, len(splits))
	for _, s := range splits {
		out = append(out, PitcherAppearance{
			Date:           s.Date,
			OpponentName:   s.Opponent.Name,
			InningsPitched: s.Stat.InningsPitched,
			Pitches:        s.Stat.NumberOfPitches,
			IsStart:        s.Stat.GamesStarted == 1,
		})
	}
	if len(out) == 0 {
		return nil, nil
	}
	// gameLog is chronological; reverse so caller sees newest first.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// FetchPitcherGameLog returns the pitcher's most recent `limit` starts
// for the given season. Returns nil,nil if the API returns no splits.
// Splits are filtered to gamesStarted == 1 (i.e. starting appearances).
func (c *Client) FetchPitcherGameLog(playerID, season, limit int) ([]GameLogStart, error) {
	url := fmt.Sprintf("%s/api/v1/people/%d/stats?stats=gameLog&group=pitching&season=%d",
		baseURL, playerID, season)

	var env gameLogEnvelope
	if err := c.fetchStat(url, &env); err != nil {
		return nil, err
	}
	if len(env.Stats) == 0 || len(env.Stats[0].Splits) == 0 {
		return nil, nil
	}
	splits := env.Stats[0].Splits
	out := make([]GameLogStart, 0, len(splits))
	for _, s := range splits {
		if s.Stat.GamesStarted != 1 {
			continue
		}
		out = append(out, GameLogStart{
			Date:           s.Date,
			OpponentName:   s.Opponent.Name,
			InningsPitched: s.Stat.InningsPitched,
			EarnedRuns:     s.Stat.EarnedRuns,
			Strikeouts:     s.Stat.StrikeOuts,
			Walks:          s.Stat.BaseOnBalls,
			HomeRuns:       s.Stat.HomeRuns,
			Pitches:        s.Stat.NumberOfPitches,
			IsStart:        true,
		})
	}
	if len(out) == 0 {
		return nil, nil
	}
	// Most recent first; gameLog returns chronological, so reverse.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
