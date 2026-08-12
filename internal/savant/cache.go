package savant

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pdavlin/go-playball/internal/reportcache"
)

// diskCache stores parsed leaderboards on disk, one file per season and
// player type, under a base directory. It mirrors the atomic-write and
// default-directory conventions of internal/reportcache but keys by
// season+type rather than gamePk, and adds a freshness TTL because
// percentile data is time-varying rather than immutable per game.
type diskCache struct {
	dir string
	ttl time.Duration
}

// cacheFile is the on-disk shape. FetchedAt drives TTL expiry; Players is
// the parsed leaderboard keyed by MLBAM player id.
type cacheFile struct {
	FetchedAt time.Time              `json:"fetched_at"`
	Year      int                    `json:"year"`
	Type      PlayerType             `json:"type"`
	Players   map[string]Percentiles `json:"players"`
}

// newDiskCache resolves the cache directory (defaulting to
// ~/.config/go-playball/savant when dir is empty) and returns a cache. A
// nil return means no writable location could be determined; callers
// treat that as "caching disabled" and always fetch.
func newDiskCache(dir string, ttl time.Duration) *diskCache {
	if dir == "" {
		resolved, err := reportcache.DefaultDir("savant")
		if err != nil {
			return nil
		}
		dir = resolved
	}
	return &diskCache{dir: dir, ttl: ttl}
}

// path returns the cache-file path for one season and player type, e.g.
// ~/.config/go-playball/savant/2026-batter.json.
func (d *diskCache) path(year int, pt PlayerType) string {
	return filepath.Join(d.dir, fmt.Sprintf("%d-%s.json", year, pt))
}

// load returns the cached players for year+type when a fresh, well-formed
// file exists. The bool is false on a miss (absent, corrupt, or expired),
// in which case the caller fetches from the network. A stale or corrupt
// file is best-effort removed.
func (d *diskCache) load(year int, pt PlayerType) (map[string]Percentiles, bool) {
	path := d.path(year, pt)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var cf cacheFile
	if err := json.Unmarshal(data, &cf); err != nil {
		_ = os.Remove(path)
		return nil, false
	}
	if time.Since(cf.FetchedAt) > d.ttl {
		return nil, false
	}
	if len(cf.Players) == 0 {
		return nil, false
	}
	return cf.Players, true
}

// save writes the parsed leaderboard atomically. Any failure is returned
// but is non-fatal to the caller, which already holds the parsed data.
func (d *diskCache) save(year int, pt PlayerType, players map[string]Percentiles) error {
	if err := os.MkdirAll(d.dir, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(cacheFile{
		FetchedAt: time.Now().UTC(),
		Year:      year,
		Type:      pt,
		Players:   players,
	})
	if err != nil {
		return err
	}

	final := d.path(year, pt)
	tmp, err := os.CreateTemp(d.dir, fmt.Sprintf(".%d-%s-*.json.tmp", year, pt))
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, final); err != nil {
		return err
	}
	cleanup = false
	return nil
}
