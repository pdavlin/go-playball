package scouting

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const cacheSchemaVersion = 2

// CacheEntry is the on-disk shape of a single cached scouting report.
type CacheEntry struct {
	SchemaVersion     int       `json:"schema_version"`
	GamePk            int       `json:"game_pk"`
	GeneratedAt       time.Time `json:"generated_at"`
	Provider          string    `json:"provider"`
	Model             string    `json:"model"`
	PromptHash        string    `json:"prompt_hash"`
	LineupFingerprint string    `json:"lineup_fingerprint,omitempty"`
	Body              string    `json:"body"`
}

// Cache stores scouting reports under a base directory keyed by gamePk.
type Cache struct {
	dir string
}

// NewCache returns a Cache rooted at dir. The directory is created lazily on
// first Save.
func NewCache(dir string) *Cache {
	return &Cache{dir: dir}
}

// DefaultCacheDir returns the platform-appropriate cache directory used by
// the go-playball binary.
func DefaultCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "go-playball", "scouting"), nil
}

func (c *Cache) path(gamePk int) string {
	return filepath.Join(c.dir, fmt.Sprintf("%d.json", gamePk))
}

// Load returns the cached entry for gamePk, or nil if absent or unusable.
// "Unusable" includes corrupted JSON and entries with a newer schema_version
// than this binary understands; in both cases Load best-effort deletes the
// bad file so the next miss writes a clean one.
func (c *Cache) Load(gamePk int) (*CacheEntry, error) {
	path := c.path(gamePk)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		_ = os.Remove(path)
		return nil, nil
	}
	if entry.SchemaVersion > cacheSchemaVersion {
		return nil, nil
	}
	return &entry, nil
}

// Save writes entry atomically: temp file in the same directory, then
// rename. Permissions are 0600 because the body can contain matchup
// commentary the user may not want world-readable.
func (c *Cache) Save(entry CacheEntry) error {
	if err := os.MkdirAll(c.dir, 0o700); err != nil {
		return err
	}
	entry.SchemaVersion = cacheSchemaVersion
	if entry.GeneratedAt.IsZero() {
		entry.GeneratedAt = time.Now().UTC()
	}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}

	final := c.path(entry.GamePk)
	tmp, err := os.CreateTemp(c.dir, fmt.Sprintf(".%d-*.json.tmp", entry.GamePk))
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

// Delete removes the cached entry for gamePk. Missing entries are not an
// error.
func (c *Cache) Delete(gamePk int) error {
	if err := os.Remove(c.path(gamePk)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
