// Package reportcache is the on-disk cache shared by scouting and recap
// report generators. Callers own their own schema version constant and
// pass it to NewCache so this package can enforce the "newer schema is a
// miss" rule without owning the version itself.
package reportcache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CacheEntry is the on-disk shape. SchemaVersion is owned by the caller.
// LineupFingerprint is scouting-specific; recap entries leave it empty
// and omit it from JSON.
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

// Cache stores entries under a base directory keyed by gamePk.
type Cache struct {
	dir            string
	maxSchemaKnown int
}

// NewCache returns a Cache rooted at dir. The directory is created lazily
// on first Save. maxSchemaKnown is the highest schema_version the caller
// understands; entries with a strictly larger version are treated as a
// miss by Load.
func NewCache(dir string, maxSchemaKnown int) *Cache {
	return &Cache{dir: dir, maxSchemaKnown: maxSchemaKnown}
}

// DefaultDir returns ~/.config/go-playball/<kind>.
func DefaultDir(kind string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "go-playball", kind), nil
}

func (c *Cache) path(gamePk int) string {
	return filepath.Join(c.dir, fmt.Sprintf("%d.json", gamePk))
}

// Load returns the cached entry for gamePk, or nil if absent or unusable.
// Corrupted JSON and entries with a newer SchemaVersion than the caller
// signaled both return nil and best-effort delete the bad file.
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
	if entry.SchemaVersion > c.maxSchemaKnown {
		return nil, nil
	}
	return &entry, nil
}

// Save writes entry atomically. SchemaVersion is forced to the caller's
// maxSchemaKnown so a stale in-memory entry cannot poison the file.
func (c *Cache) Save(entry CacheEntry) error {
	if err := os.MkdirAll(c.dir, 0o700); err != nil {
		return err
	}
	entry.SchemaVersion = c.maxSchemaKnown
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
