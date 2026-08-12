package ui

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/pdavlin/go-playball/internal/api"
)

func TestHotBatsWindowDefaults(t *testing.T) {
	// Zero value should default to L7 for both gamesInWindow and label.
	var w HotBatsWindow
	if got := w.gamesInWindow(); got != 7 {
		t.Errorf("default window games = %d, want 7", got)
	}
	if got := w.label(); got != "L7" {
		t.Errorf("default window label = %q, want L7", got)
	}
	if got := paMinForWindow(w); got != 15 {
		t.Errorf("default PA min = %d, want 15", got)
	}
}

func TestHotBatsWindowMapping(t *testing.T) {
	cases := []struct {
		w     HotBatsWindow
		games int
		paMin int
		label string
	}{
		{HotBatsL7, 7, 15, "L7"},
		{HotBatsL15, 15, 35, "L15"},
		{HotBatsL30, 30, 70, "L30"},
	}
	for _, tc := range cases {
		if got := tc.w.gamesInWindow(); got != tc.games {
			t.Errorf("%s gamesInWindow = %d, want %d", tc.label, got, tc.games)
		}
		if got := paMinForWindow(tc.w); got != tc.paMin {
			t.Errorf("%s paMin = %d, want %d", tc.label, got, tc.paMin)
		}
		if got := tc.w.label(); got != tc.label {
			t.Errorf("label = %q, want %q", got, tc.label)
		}
	}
}

func TestParseStatFloat(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{".567", 0.567},
		{"1.506", 1.506},
		{"", 0},
		{".---", 0}, // malformed sentinel returned by API for no-AB games
	}
	for _, tc := range cases {
		got := parseStatFloat(tc.in)
		if got != tc.want {
			t.Errorf("parseStatFloat(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestRenderHotBatsRowsEmpty(t *testing.T) {
	out := renderHotBatsRows(nil)
	if !strings.Contains(out, "Insufficient data") {
		t.Errorf("expected insufficient-data fallback, got: %q", out)
	}
}

func TestRenderHotBatsRowsFormatting(t *testing.T) {
	rows := []hotBatsRow{
		{name: "Aaron Judge", avg: ".325", homeRuns: 4, rbi: 11, ops: "1.205", pa: 32},
	}
	out := renderHotBatsRows(rows)
	for _, want := range []string{"Aaron Judge", ".325", "1.205", "11"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q, got: %q", want, out)
		}
	}
}

func TestRenderHotBatsRowsPartialFallback(t *testing.T) {
	rows := []hotBatsRow{
		{name: "A", avg: ".300", homeRuns: 1, rbi: 2, ops: ".850", pa: 20},
	}
	out := renderHotBatsRows(rows)
	if !strings.Contains(out, "insufficient data for remaining slots") {
		t.Errorf("expected remaining-slots notice with 1 row, got: %q", out)
	}
}

func TestRenderHotBatsTeamErrorFallback(t *testing.T) {
	payload := &hotBatsTeamPayload{err: errors.New("net down"), loaded: true}
	out := renderHotBatsTeam("HOU", TeamColors{}, payload, nil)
	if !strings.Contains(out, "Data unavailable") {
		t.Errorf("expected error fallback, got: %q", out)
	}
}

func TestRenderHotBatsTeamSkeleton(t *testing.T) {
	// payload nil → skeleton path.
	out := renderHotBatsTeam("HOU", TeamColors{}, nil, nil)
	for _, want := range []string{"HOU", "Name", "AVG", "HR", "RBI", "OPS"} {
		if !strings.Contains(out, want) {
			t.Errorf("skeleton missing %q, got: %q", want, out)
		}
	}
}

func TestRenderHotBatsHeader(t *testing.T) {
	out := renderHotBatsHeader(HotBatsL15, 80, lipgloss.Color("#123456"))
	// All three labels should appear; active one bracketed.
	for _, want := range []string{"L7", "L15", "L30", "[ L15 ]"} {
		if !strings.Contains(out, want) {
			t.Errorf("header missing %q, got: %q", want, out)
		}
	}
}

func TestPickHotBatsTeamsNilSafe(t *testing.T) {
	a, h := pickHotBatsTeams(nil, HotBatsL7)
	if a != nil || h != nil {
		t.Error("nil data should return nil/nil")
	}
	a, h = pickHotBatsTeams(&pregameGameData{}, HotBatsL7)
	if a != nil || h != nil {
		t.Error("empty hotBats should return nil/nil")
	}
}

func TestComputeHotBatsRankingTieBreak(t *testing.T) {
	// Exercise the sort independently by piping rows through the same
	// comparator computeHotBatsTeam uses. Two rows with equal OPS must
	// break by HR; with equal HR, by AVG.
	got := sortHotBatsRowsForTest([]hotBatsRow{
		{name: "lo-hr", ops: "1.000", homeRuns: 1, avg: ".300"},
		{name: "hi-hr", ops: "1.000", homeRuns: 3, avg: ".280"},
		{name: "tie-hr-hi-avg", ops: "1.000", homeRuns: 3, avg: ".320"},
		{name: "best-ops", ops: "1.200", homeRuns: 0, avg: ".250"},
	})
	want := []string{"best-ops", "tie-hr-hi-avg", "hi-hr", "lo-hr"}
	for i, w := range want {
		if got[i].name != w {
			t.Errorf("rank[%d] = %s, want %s (full order: %v)", i, got[i].name, w, names(got))
		}
	}
}

func TestHotBatsPAThresholdFilter(t *testing.T) {
	// At L7 the PA threshold is 15. Rows below 15 should be dropped.
	rows := filterHotBatsByPAForTest(
		[]hotBatsRow{
			{name: "above", pa: 16},
			{name: "below", pa: 14},
			{name: "exactly", pa: 15},
		},
		paMinForWindow(HotBatsL7),
	)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows after PA filter, got %d", len(rows))
	}
	for _, r := range rows {
		if r.name == "below" {
			t.Errorf("below-threshold row should have been filtered")
		}
	}
}

// Helpers under test ----------------------------------------------------

// sortHotBatsRowsForTest mirrors the sort logic used in
// computeHotBatsTeam so the comparator is reachable from tests without
// instantiating the live API client.
func sortHotBatsRowsForTest(rows []hotBatsRow) []hotBatsRow {
	out := make([]hotBatsRow, len(rows))
	copy(out, rows)
	sortHotBatsRows(out)
	return out
}

func filterHotBatsByPAForTest(rows []hotBatsRow, paMin int) []hotBatsRow {
	out := rows[:0]
	for _, r := range rows {
		if r.pa < paMin {
			continue
		}
		out = append(out, r)
	}
	return out
}

func names(rows []hotBatsRow) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.name
	}
	return out
}

// silence unused-imports if a test is later commented out
var _ = api.RosterPlayer{}
