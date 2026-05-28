package scouting

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCache_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir)

	entry := CacheEntry{
		GamePk:     777,
		Provider:   "anthropic",
		Model:      "claude-test",
		PromptHash: "abc",
		Body:       "## The Edge\nLAD by a sliver.\n",
	}
	if err := c.Save(entry); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := c.Load(777)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got == nil {
		t.Fatal("Load returned nil for present entry")
	}
	if got.Body != entry.Body || got.PromptHash != entry.PromptHash {
		t.Errorf("round-trip mismatch: %+v", got)
	}
	if got.GeneratedAt.IsZero() {
		t.Error("GeneratedAt was not populated")
	}
}

func TestCache_LoadAbsent(t *testing.T) {
	c := NewCache(t.TempDir())
	got, err := c.Load(123)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != nil {
		t.Errorf("want nil for missing gamePk, got %+v", got)
	}
}

func TestCache_NewerSchemaIgnored(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir)
	if err := c.Save(CacheEntry{GamePk: 42, Body: "x"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// Manually bump schema_version on disk.
	path := filepath.Join(dir, "42.json")
	data, _ := os.ReadFile(path)
	// Cheap rewrite: replace schema_version value.
	bumped := []byte(`{"schema_version":99,"game_pk":42,"body":"x"}`)
	if err := os.WriteFile(path, bumped, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := c.Load(42)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != nil {
		t.Errorf("want nil for newer schema, got %+v", got)
	}
	_ = data
}

func TestCache_AtomicWriteNoOrphan(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir)
	if err := c.Save(CacheEntry{GamePk: 1, Body: "y"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if e.Name() != "1.json" {
			t.Errorf("unexpected leftover file: %s", e.Name())
		}
	}
}

func TestCache_Delete(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir)
	_ = c.Save(CacheEntry{GamePk: 5, Body: "b"})
	if err := c.Delete(5); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	// Delete of missing entry is not an error.
	if err := c.Delete(5); err != nil {
		t.Fatalf("Delete missing: %v", err)
	}
}

func TestCache_CorruptedFileTreatedAsMiss(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir)
	path := filepath.Join(dir, "9.json")
	_ = os.MkdirAll(dir, 0o700)
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := c.Load(9)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != nil {
		t.Errorf("want nil for corrupt file, got %+v", got)
	}
}

func TestCache_LineupFingerprintRoundTrip(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir)
	entry := CacheEntry{
		GamePk:            7,
		Provider:          "anthropic",
		Model:             "claude-test",
		PromptHash:        "deadbeef",
		LineupFingerprint: "away:1,2,3|home:10,11,12",
		Body:              "## The Edge\n",
	}
	if err := c.Save(entry); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := c.Load(7)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got == nil {
		t.Fatal("nil entry")
	}
	if got.SchemaVersion != cacheSchemaVersion {
		t.Errorf("schema_version = %d, want %d", got.SchemaVersion, cacheSchemaVersion)
	}
	if got.LineupFingerprint != entry.LineupFingerprint {
		t.Errorf("fingerprint = %q, want %q", got.LineupFingerprint, entry.LineupFingerprint)
	}
}

func TestCache_LoadsV1ThenOverwritesAsV2(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir)
	path := filepath.Join(dir, "33.json")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	v1 := []byte(`{"schema_version":1,"game_pk":33,"prompt_hash":"old","body":"old body"}`)
	if err := os.WriteFile(path, v1, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := c.Load(33)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got == nil || got.SchemaVersion != 1 || got.PromptHash != "old" {
		t.Fatalf("expected v1 entry to load, got %+v", got)
	}

	if err := c.Save(CacheEntry{
		GamePk:            33,
		PromptHash:        "new",
		LineupFingerprint: "away:1|home:2",
		Body:              "new body",
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got2, err := c.Load(33)
	if err != nil {
		t.Fatalf("Load 2: %v", err)
	}
	if got2 == nil {
		t.Fatal("nil after overwrite")
	}
	if got2.SchemaVersion != cacheSchemaVersion {
		t.Errorf("schema_version after overwrite = %d, want %d", got2.SchemaVersion, cacheSchemaVersion)
	}
	if got2.PromptHash != "new" || got2.Body != "new body" {
		t.Errorf("v1 file not overwritten: %+v", got2)
	}
}

// Sanity: GeneratedAt round-trips at second-ish precision through JSON.
func TestCache_GeneratedAtPreserved(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir)
	at := time.Date(2026, 5, 27, 18, 30, 11, 0, time.UTC)
	if err := c.Save(CacheEntry{GamePk: 11, Body: "z", GeneratedAt: at}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, _ := c.Load(11)
	if got == nil {
		t.Fatal("nil entry")
	}
	if !got.GeneratedAt.Equal(at) {
		t.Errorf("want %v, got %v", at, got.GeneratedAt)
	}
}
