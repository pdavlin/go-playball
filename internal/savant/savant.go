// Package savant fetches Baseball Savant (Statcast) percentile-ranking
// leaderboards and exposes them keyed by MLBAM player id. It is a
// standalone data source: the scouting package joins its output onto the
// probable starters and lineup batters it already assembles from the MLB
// Stats API.
//
// The upstream endpoint returns an HTML page, not JSON. The per-player
// data is embedded in the markup as a JavaScript assignment,
// `var leaderboard_data = [ ... ];`. We extract that array with a regex
// and JSON-decode it (see parse.go).
//
// Every network and parse failure is recoverable: callers receive an
// error and are expected to degrade to their non-Savant behavior rather
// than abort. Nothing here panics on bad input.
package savant

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// PlayerType selects which percentile leaderboard to fetch. Savant serves
// batters and pitchers as separate pages.
type PlayerType string

const (
	// Batter is the hitting percentile leaderboard (type=batter).
	Batter PlayerType = "batter"
	// Pitcher is the pitching percentile leaderboard (type=pitcher).
	Pitcher PlayerType = "pitcher"
)

const (
	// savantHost is the Baseball Savant origin. Percentile rankings live
	// under /leaderboard/percentile-rankings.
	savantHost = "https://baseballsavant.mlb.com"
	// fetchTimeout bounds a single leaderboard fetch. The HTML payload is
	// ~900KB, so this is deliberately looser than the Stats API client.
	fetchTimeout = 20 * time.Second
	// browserUserAgent is sent on every request. A bare Go User-Agent is
	// liable to draw a 403 from the Savant edge, so we present as a
	// desktop browser.
	browserUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36"
	// defaultCacheTTL is how long a cached leaderboard is considered
	// fresh. Percentile ranks move slowly over a season, so a day avoids
	// re-pulling ~900KB on every report while staying current enough.
	defaultCacheTTL = 24 * time.Hour
)

// Percentiles is the subset of Savant percentile ranks the scouting
// report uses. Every field is 0-100 where a HIGHER percentile is better
// for the player (Savant already flips "lower is better" metrics such as
// K% and chase% so that up is always good). A nil pointer means the
// player carried a null for that metric (typically low sample), so the
// consumer omits it rather than inventing a value.
//
// One struct serves both player types; batters populate the hitting
// fields and pitchers populate the pitching fields, with the shared plate
// -discipline and contact-quality ranks meaningful for both. The joins in
// the scouting package decide which subset to render.
type Percentiles struct {
	// PlayerID is the MLBAM id as a string, matching the Stats API player
	// ids once converted. It is the join key.
	PlayerID string `json:"player_id"`
	// Name is "Last, First" as Savant reports it. Kept for debugging; the
	// scouting layer uses its own names.
	Name string `json:"player_name"`

	// Hitting-side expected vs actual. The gap between an expected rank
	// (xwOBA, xBA, xSLG) and its actual counterpart (wOBA, SLG) is the
	// expected-vs-actual story.
	XWOBAPct *int `json:"percent_rank_xwoba"`
	WOBAPct  *int `json:"percent_rank_woba"`
	XBAPct   *int `json:"percent_rank_xba"`
	XSLGPct  *int `json:"percent_rank_xslg"`
	SLGPct   *int `json:"percent_rank_slg"`

	// Contact quality (shared vocabulary; for pitchers these are the
	// contact allowed).
	HardHitPct *int `json:"percent_rank_hard_hit_percent"`
	BarrelPct  *int `json:"percent_rank_barrel_batted_rate"`

	// Plate discipline (shared).
	KPct     *int `json:"percent_rank_k_percent"`
	BBPct    *int `json:"percent_rank_bb_percent"`
	WhiffPct *int `json:"percent_rank_whiff_percent"`
	ChasePct *int `json:"percent_rank_chase_percent"`

	// Hitting athleticism.
	SprintSpeedPct *int `json:"percent_speed_order"`

	// Pitching-side expected run prevention and stuff.
	XERAPct         *int `json:"percent_rank_xera"`
	FastballVeloPct *int `json:"percent_rank_fastball_velo"`
}

// Client fetches Savant leaderboards with an on-disk cache. The zero
// value is not usable; call NewClient.
type Client struct {
	httpClient *http.Client
	cache      *diskCache
}

// NewClient returns a Client that caches parsed leaderboards under
// cacheDir. When cacheDir is empty the default location
// (~/.config/go-playball/savant) is used; if even that cannot be
// resolved, the client still works but skips disk caching.
func NewClient(cacheDir string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: fetchTimeout},
		cache:      newDiskCache(cacheDir, defaultCacheTTL),
	}
}

// Rankings returns the percentile map for a season and player type, keyed
// by MLBAM player id. It serves a fresh disk-cache hit without any
// network call; otherwise it fetches, parses, caches, and returns. A
// non-nil error means no usable data was obtained and the caller should
// degrade gracefully.
func (c *Client) Rankings(year int, pt PlayerType) (map[string]Percentiles, error) {
	if c.cache != nil {
		if players, ok := c.cache.load(year, pt); ok {
			return players, nil
		}
	}

	html, err := c.fetch(year, pt)
	if err != nil {
		return nil, err
	}
	players, err := ParseLeaderboardData(html)
	if err != nil {
		return nil, err
	}
	if c.cache != nil {
		// A cache-write failure is non-fatal: return the parsed data
		// regardless so a read-only home dir never breaks the report.
		_ = c.cache.save(year, pt, players)
	}
	return players, nil
}

// fetch performs the HTTP GET against the percentile-rankings endpoint
// and returns the raw HTML body.
func (c *Client) fetch(year int, pt PlayerType) (string, error) {
	url := fmt.Sprintf("%s/leaderboard/percentile-rankings?type=%s&year=%d",
		savantHost, pt, year)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("savant: building request: %w", err)
	}
	req.Header.Set("User-Agent", browserUserAgent)
	req.Header.Set("Accept", "text/html")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("savant: fetching %s rankings: %w", pt, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("savant: %s rankings returned status %d", pt, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("savant: reading %s rankings body: %w", pt, err)
	}
	return string(body), nil
}
