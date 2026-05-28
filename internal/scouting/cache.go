package scouting

import "github.com/pdavlin/go-playball/internal/reportcache"

// cacheSchemaVersion is the scouting-side cache format version. It bumps
// when the prompt hash inputs or stored fields change in a way that
// makes prior entries logically stale.
const cacheSchemaVersion = 2

// Cache and CacheEntry are aliased from reportcache so existing call
// sites (scouting.Cache, scouting.CacheEntry) keep working without
// importing reportcache directly.
type Cache = reportcache.Cache
type CacheEntry = reportcache.CacheEntry

// NewCache returns a scouting cache rooted at dir.
func NewCache(dir string) *Cache {
	return reportcache.NewCache(dir, cacheSchemaVersion)
}

// DefaultCacheDir returns the platform-appropriate cache directory used
// by the go-playball binary for scouting reports.
func DefaultCacheDir() (string, error) {
	return reportcache.DefaultDir("scouting")
}
