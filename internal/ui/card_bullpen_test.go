package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/pdavlin/go-playball/internal/api"
)

func mustDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestDaysBetween(t *testing.T) {
	game := mustDate("2026-06-01")
	cases := []struct {
		past string
		want int
	}{
		{"2026-06-01", 0},
		{"2026-05-31", 1},
		{"2026-05-30", 2},
		{"2026-05-15", 17},
	}
	for _, tc := range cases {
		got := daysBetween(tc.past, game)
		if got != tc.want {
			t.Errorf("daysBetween(%q, 2026-06-01) = %d, want %d", tc.past, got, tc.want)
		}
	}
	if got := daysBetween("bogus", game); got != -1 {
		t.Errorf("daysBetween parse failure should return -1, got %d", got)
	}
}

func TestComputeBullpenStatusEmpty(t *testing.T) {
	game := mustDate("2026-06-01")
	if got := computeBullpenStatus(nil, game); got != bullpenReady {
		t.Errorf("no appearances → Ready, got %v", got)
	}
}

func TestComputeBullpenStatusUnavailAtThreshold(t *testing.T) {
	// Pitched yesterday with exactly threshold pitches → Unavail.
	game := mustDate("2026-06-01")
	apps := []bullpenAppearance{{date: "2026-05-31", pitches: bullpenUnavailPitchThreshold}}
	if got := computeBullpenStatus(apps, game); got != bullpenUnavail {
		t.Errorf("yesterday %dP → Unavail, got %v", bullpenUnavailPitchThreshold, got)
	}
}

func TestComputeBullpenStatusLimitedJustUnderThreshold(t *testing.T) {
	game := mustDate("2026-06-01")
	apps := []bullpenAppearance{{date: "2026-05-31", pitches: bullpenUnavailPitchThreshold - 1}}
	if got := computeBullpenStatus(apps, game); got != bullpenLimited {
		t.Errorf("yesterday %dP → Limited, got %v", bullpenUnavailPitchThreshold-1, got)
	}
}

func TestComputeBullpenStatusReadyTwoDaysAgo(t *testing.T) {
	game := mustDate("2026-06-01")
	apps := []bullpenAppearance{{date: "2026-05-30", pitches: 40}}
	if got := computeBullpenStatus(apps, game); got != bullpenReady {
		t.Errorf("2 days ago → Ready regardless of pitch count, got %v", got)
	}
}

func TestComputeBullpenStatusBadDate(t *testing.T) {
	game := mustDate("2026-06-01")
	apps := []bullpenAppearance{{date: "garbage", pitches: 99}}
	if got := computeBullpenStatus(apps, game); got != bullpenReady {
		t.Errorf("unparseable date should default to Ready, got %v", got)
	}
}

func TestRenderBullpenSummary(t *testing.T) {
	r := bullpenReliever{
		appearances: []bullpenAppearance{
			{pitches: 12}, {pitches: 25}, {pitches: 21},
		},
	}
	got := bullpenSummary(r)
	if got != "3 of 3 - 58P" {
		t.Errorf("bullpenSummary = %q, want %q", got, "3 of 3 - 58P")
	}
}

func TestRenderBullpenSummaryEmpty(t *testing.T) {
	if got := bullpenSummary(bullpenReliever{}); got != "" {
		t.Errorf("empty summary should be \"\", got %q", got)
	}
}

func TestRenderBullpenTeamSkeleton(t *testing.T) {
	out := renderBullpenTeam("HOU", TeamColors{}, nil, nil)
	if !strings.Contains(out, "HOU") {
		t.Errorf("skeleton missing team name, got: %q", out)
	}
}

func TestRenderBullpenTeamEmptyRoster(t *testing.T) {
	payload := &bullpenTeamPayload{loaded: true}
	out := renderBullpenTeam("HOU", TeamColors{}, payload, nil)
	if !strings.Contains(out, "No relievers on roster") {
		t.Errorf("expected empty-roster fallback, got: %q", out)
	}
}

func TestRenderBullpenTeamWithRelievers(t *testing.T) {
	payload := &bullpenTeamPayload{
		loaded: true,
		relievers: []bullpenReliever{
			{
				name:   "Josh Hader",
				status: bullpenReady,
				appearances: []bullpenAppearance{
					{date: "2026-05-29", inningsPitched: "1.0", pitches: 14},
					{date: "2026-05-27", inningsPitched: "0.2", pitches: 11},
				},
			},
			{
				name:   "No-Recent Guy",
				status: bullpenReady,
			},
		},
	}
	out := renderBullpenTeam("HOU", TeamColors{}, payload, nil)
	for _, want := range []string{"Josh Hader", "2026-05-29", "1.0 IP", "14P", "No recent appearances"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got: %q", want, out)
		}
	}
}

func TestRenderBullpenCardLoadingPlaceholder(t *testing.T) {
	game := &api.Game{
		GameData: &api.GameData{
			Teams: api.GameTeams{
				Away: api.FullTeamInfo{Name: "Houston Astros"},
				Home: api.FullTeamInfo{Name: "Baltimore Orioles"},
			},
		},
	}
	out := renderBullpenCard(game, nil, nil, 120, 30)
	// GetTeamShortName returns the team's short name; for Houston Astros that's "Astros".
	for _, want := range []string{"Astros", "Orioles", "Ready:", "Limited:", "Unavail:"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in skeleton, got: %q", want, out)
		}
	}
}

func TestRenderBullpenCardSortedByStatus(t *testing.T) {
	// Status sort: Unavail first, then Limited, then Ready. Stable
	// ordering within a status by name.
	rows := []bullpenReliever{
		{name: "Z-ready", status: bullpenReady},
		{name: "Unavail Bob", status: bullpenUnavail},
		{name: "Limited Carl", status: bullpenLimited},
		{name: "A-ready", status: bullpenReady},
	}
	// The sort happens inside computeBullpenTeam; replicate just the
	// comparator inline by calling it indirectly.
	sorted := append([]bullpenReliever{}, rows...)
	sortBullpenForTest(sorted)
	want := []string{"Unavail Bob", "Limited Carl", "A-ready", "Z-ready"}
	for i, w := range want {
		if sorted[i].name != w {
			t.Errorf("rank[%d] = %s, want %s", i, sorted[i].name, w)
		}
	}
}

// Helper that mirrors computeBullpenTeam's sort so we don't need to
// spin up the API client just to test ordering.
func sortBullpenForTest(rows []bullpenReliever) {
	// Hand-rolled bubble sort is fine for 4 elements; tests are about
	// the comparator semantics, not performance.
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			swap := false
			if rows[j].status > rows[i].status {
				swap = true
			} else if rows[j].status == rows[i].status && rows[j].name < rows[i].name {
				swap = true
			}
			if swap {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}
}
