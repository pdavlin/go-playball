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
