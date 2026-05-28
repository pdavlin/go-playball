package reportcache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const testMaxSchema = 2

func TestCache_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir, testMaxSchema)

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
	if got.SchemaVersion != testMaxSchema {
		t.Errorf("SchemaVersion = %d, want %d", got.SchemaVersion, testMaxSchema)
	}
}

func TestCache_LoadAbsent(t *testing.T) {
	c := NewCache(t.TempDir(), testMaxSchema)
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
	c := NewCache(dir, testMaxSchema)
	if err := c.Save(CacheEntry{GamePk: 42, Body: "x"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	path := filepath.Join(dir, "42.json")
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
}

func TestCache_AtomicWriteNoOrphan(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir, testMaxSchema)
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
	c := NewCache(dir, testMaxSchema)
	_ = c.Save(CacheEntry{GamePk: 5, Body: "b"})
	if err := c.Delete(5); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := c.Delete(5); err != nil {
		t.Fatalf("Delete missing: %v", err)
	}
}

func TestCache_CorruptedFileTreatedAsMiss(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir, testMaxSchema)
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

func TestCache_GeneratedAtPreserved(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir, testMaxSchema)
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

func TestDefaultDir_JoinsKind(t *testing.T) {
	dir, err := DefaultDir("scouting")
	if err != nil {
		t.Fatalf("DefaultDir: %v", err)
	}
	if filepath.Base(dir) != "scouting" {
		t.Errorf("kind not last segment: %s", dir)
	}
	if filepath.Base(filepath.Dir(dir)) != "go-playball" {
		t.Errorf("missing go-playball parent: %s", dir)
	}
}
