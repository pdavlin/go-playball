package scouting

import "testing"

func TestLineupFingerprint_Deterministic(t *testing.T) {
	away := LineupCtx{Batters: []BatterCtx{
		{PlayerID: 1}, {PlayerID: 2}, {PlayerID: 3},
	}}
	home := LineupCtx{Batters: []BatterCtx{
		{PlayerID: 10}, {PlayerID: 11}, {PlayerID: 12},
	}}
	first := lineupFingerprint(away, home)
	second := lineupFingerprint(away, home)
	if first != second {
		t.Errorf("fingerprint not stable: %q vs %q", first, second)
	}
	if first != "away:1,2,3|home:10,11,12" {
		t.Errorf("unexpected fingerprint: %q", first)
	}
}

func TestLineupFingerprint_SwapDetected(t *testing.T) {
	away := LineupCtx{Batters: []BatterCtx{
		{PlayerID: 1}, {PlayerID: 2}, {PlayerID: 3},
	}}
	home := LineupCtx{Batters: []BatterCtx{
		{PlayerID: 10}, {PlayerID: 11}, {PlayerID: 12},
	}}
	swapped := LineupCtx{Batters: []BatterCtx{
		{PlayerID: 2}, {PlayerID: 1}, {PlayerID: 3},
	}}
	if lineupFingerprint(away, home) == lineupFingerprint(swapped, home) {
		t.Error("expected fingerprint to change when batting order swaps")
	}
}

func TestLineupFingerprint_EmptyLineup(t *testing.T) {
	got := lineupFingerprint(LineupCtx{}, LineupCtx{})
	if got != "away:|home:" {
		t.Errorf("empty fingerprint = %q, want %q", got, "away:|home:")
	}
}
