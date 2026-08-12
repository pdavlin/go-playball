package savant

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// leaderboardRe captures the JSON array assigned to `leaderboard_data` in
// the Savant HTML page. The value is a single JSON array literal
// terminated by `];`. A non-greedy body match stops at the first `];`,
// which is the array's real terminator because no metric value in the
// payload contains that sequence.
var leaderboardRe = regexp.MustCompile(`leaderboard_data\s*=\s*(\[[\s\S]*?\])\s*;`)

// ParseLeaderboardData extracts the embedded `leaderboard_data` array from
// a Savant percentile-rankings HTML page and returns the per-player
// percentiles keyed by MLBAM player id.
//
// Entries whose player_id is empty are skipped. Null metric values decode
// to nil pointers on Percentiles, so low-sample players simply carry
// fewer populated fields. A missing array or malformed JSON is returned
// as an error so the caller can degrade rather than proceed on garbage.
func ParseLeaderboardData(html string) (map[string]Percentiles, error) {
	m := leaderboardRe.FindStringSubmatch(html)
	if len(m) < 2 {
		return nil, fmt.Errorf("savant: leaderboard_data array not found in page")
	}

	var entries []Percentiles
	if err := json.Unmarshal([]byte(m[1]), &entries); err != nil {
		return nil, fmt.Errorf("savant: decoding leaderboard_data: %w", err)
	}

	out := make(map[string]Percentiles, len(entries))
	for _, e := range entries {
		if e.PlayerID == "" {
			continue
		}
		out[e.PlayerID] = e
	}
	return out, nil
}
