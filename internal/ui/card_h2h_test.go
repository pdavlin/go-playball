package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pdavlin/go-playball/internal/api"
)

// mkGame builds a Final-state schedule entry between the two teams.
func mkGame(date string, awayID, awayRuns, homeID, homeRuns int) api.Game {
	t, _ := time.Parse("2006-01-02", date)
	return api.Game{
		GameDate: t,
		Status:   api.GameStatus{AbstractGameState: "Final"},
		Teams: api.Teams{
			Away: api.TeamInfo{Team: api.Team{ID: awayID}, Score: awayRuns},
			Home: api.TeamInfo{Team: api.Team{ID: homeID}, Score: homeRuns},
		},
	}
}

func TestAggregateH2HEmpty(t *testing.T) {
	got := aggregateH2H(nil, 1, 2)
	if !got.loaded {
		t.Error("aggregateH2H should mark payload loaded")
	}
	if got.games != 0 {
		t.Errorf("games = %d, want 0", got.games)
	}
	if got.lastMeeting != nil {
		t.Error("lastMeeting should be nil for empty input")
	}
}

func TestAggregateH2HBasic(t *testing.T) {
	// Current matchup: team 100 is the away team, team 200 is the home
	// team. Two past meetings, one in each home/away alignment.
	//   g1: 100@200, 100 wins 5-3 (current-away wins by 2).
	//   g2: 200@100, 200 wins 6-1 (current-home wins by 5).
	games := []api.Game{
		mkGame("2026-04-10", 100, 5, 200, 3),
		mkGame("2026-04-25", 200, 6, 100, 1),
	}
	got := aggregateH2H(games, 100, 200)
	if got.games != 2 {
		t.Fatalf("games = %d, want 2", got.games)
	}
	if got.awayWins != 1 || got.homeWins != 1 {
		t.Errorf("wins = %d-%d, want 1-1", got.awayWins, got.homeWins)
	}
	// Normalized totals from current-away perspective:
	//   g1: currentAway=5, currentHome=3
	//   g2 (swapped): currentAway=pastHome=1, currentHome=pastAway=6
	//   sums: away=6, home=9
	if got.awayRunsTotal != 6 || got.homeRunsTotal != 9 {
		t.Errorf("totals = %d-%d, want 6-9", got.awayRunsTotal, got.homeRunsTotal)
	}
	if got.oneRunGames != 0 {
		t.Errorf("oneRunGames = %d, want 0", got.oneRunGames)
	}
	if got.largestMargin != 5 {
		t.Errorf("largestMargin = %d, want 5", got.largestMargin)
	}
	if got.lastMeeting == nil || got.lastMeeting.date != "2026-04-25" {
		t.Errorf("lastMeeting wrong: %+v", got.lastMeeting)
	}
	// Last meeting normalization (g2 swapped): currentAway=1, currentHome=6.
	if got.lastMeeting.awayRuns != 1 || got.lastMeeting.homeRuns != 6 {
		t.Errorf("lastMeeting score = %d-%d, want 1-6",
			got.lastMeeting.awayRuns, got.lastMeeting.homeRuns)
	}
}

func TestAggregateH2HSkipsUnrelatedGames(t *testing.T) {
	// A defensive case: schedule shouldn't return unrelated games, but
	// if it does, the aggregator must skip them.
	games := []api.Game{
		mkGame("2026-04-10", 100, 5, 999, 3), // 999 not in matchup
	}
	got := aggregateH2H(games, 100, 200)
	if got.games != 0 {
		t.Errorf("games = %d, want 0 (unrelated game should be skipped)", got.games)
	}
}

func TestAggregateH2HLargestMargin(t *testing.T) {
	games := []api.Game{
		mkGame("2026-04-10", 100, 1, 200, 0),  // margin 1
		mkGame("2026-04-15", 100, 10, 200, 2), // margin 8
		mkGame("2026-04-20", 100, 3, 200, 5),  // margin 2
	}
	got := aggregateH2H(games, 100, 200)
	if got.largestMargin != 8 {
		t.Errorf("largestMargin = %d, want 8", got.largestMargin)
	}
	if got.oneRunGames != 1 {
		t.Errorf("oneRunGames = %d, want 1", got.oneRunGames)
	}
}

func TestRenderH2HCardLoadingSkeleton(t *testing.T) {
	game := &api.Game{
		GameData: &api.GameData{
			Teams: api.GameTeams{
				Away: api.FullTeamInfo{Name: "Houston Astros"},
				Home: api.FullTeamInfo{Name: "Baltimore Orioles"},
			},
		},
	}
	out := renderH2HCard(game, nil, nil, 100)
	for _, want := range []string{"HOU", "BAL", "Series", "Last meeting", "Runs", "Avg score"} {
		if !strings.Contains(out, want) {
			t.Errorf("skeleton missing %q, got: %q", want, out)
		}
	}
}

func TestRenderH2HCardNoMeetings(t *testing.T) {
	game := &api.Game{
		GameData: &api.GameData{
			Teams: api.GameTeams{
				Away: api.FullTeamInfo{Name: "Houston Astros"},
				Home: api.FullTeamInfo{Name: "Baltimore Orioles"},
			},
		},
	}
	data := &pregameGameData{h2h: &h2hPayload{loaded: true}}
	out := renderH2HCard(game, data, nil, 100)
	if !strings.Contains(out, "No prior meetings this season") {
		t.Errorf("expected no-meetings fallback, got: %q", out)
	}
}

func TestRenderH2HCardErrorFallback(t *testing.T) {
	game := &api.Game{
		GameData: &api.GameData{
			Teams: api.GameTeams{
				Away: api.FullTeamInfo{Name: "Houston Astros"},
				Home: api.FullTeamInfo{Name: "Baltimore Orioles"},
			},
		},
	}
	data := &pregameGameData{h2h: &h2hPayload{loaded: true, err: errors.New("net")}}
	out := renderH2HCard(game, data, nil, 100)
	if !strings.Contains(out, "Series data unavailable") {
		t.Errorf("expected error fallback, got: %q", out)
	}
}

func TestRenderH2HBipolarBarOneSided(t *testing.T) {
	// KC 0 - 2 LAD: away shut out. The bar must still visibly read as
	// a ratio (split marker + dim sliver), not a solid unlabeled block.
	p := &h2hPayload{games: 2, awayWins: 0, homeWins: 2}
	out := renderH2HBipolarBar(p, TeamColors{}, TeamColors{}, 40)

	if !strings.Contains(out, "│") {
		t.Fatalf("expected a split marker in a one-sided bar, got: %q", out)
	}
	markerIdx := strings.Index(out, "│")
	awaySide := out[:markerIdx]
	homeSide := out[markerIdx+len("│"):]

	awayBlocks := strings.Count(awaySide, "█")
	homeBlocks := strings.Count(homeSide, "█")
	if awayBlocks == 0 {
		t.Error("shut-out side should still render a dim sliver, got none")
	}
	if awayBlocks >= homeBlocks {
		t.Errorf("shut-out side (%d) should be far smaller than the winning side (%d)", awayBlocks, homeBlocks)
	}
}

func TestRenderH2HBipolarBarBalanced(t *testing.T) {
	p := &h2hPayload{games: 4, awayWins: 2, homeWins: 2}
	out := renderH2HBipolarBar(p, TeamColors{}, TeamColors{}, 40)
	if strings.Count(out, "│") != 1 {
		t.Errorf("expected exactly one split marker, got %d in %q", strings.Count(out, "│"), out)
	}
}

// TestRenderH2HStatGridComposesHorizontally guards against the grid
// collapsing back into vertically-concatenated cells (each cell is a
// "label\nvalue" block; row assembly must join them side by side with
// lipgloss.JoinHorizontal, not string-concatenate them).
func TestRenderH2HStatGridComposesHorizontally(t *testing.T) {
	p := &h2hPayload{
		games:         5,
		awayRunsTotal: 22,
		homeRunsTotal: 18,
		oneRunGames:   2,
		largestMargin: 6,
	}
	out := renderH2HStatGrid(p, "AWY", "HOM")
	lines := strings.Split(out, "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines (2 rows x label/value), got %d: %q", len(lines), lines)
	}

	row1Labels, row1Values := lines[0], lines[1]
	row2Labels, row2Values := lines[2], lines[3]

	for _, want := range []string{"Runs", "Run diff", "Avg score"} {
		if !strings.Contains(row1Labels, want) {
			t.Errorf("row1 labels missing %q, got: %q", want, row1Labels)
		}
	}
	for _, unwanted := range []string{"Avg total", "1-run games", "Largest margin"} {
		if strings.Contains(row1Labels, unwanted) {
			t.Errorf("row1 labels should not contain row2 label %q, got: %q", unwanted, row1Labels)
		}
	}

	for _, want := range []string{"22-18", "+4 AWY", "4.4-3.6"} {
		if !strings.Contains(row1Values, want) {
			t.Errorf("row1 values missing %q, got: %q", want, row1Values)
		}
	}

	for _, want := range []string{"Avg total", "1-run games", "Largest margin"} {
		if !strings.Contains(row2Labels, want) {
			t.Errorf("row2 labels missing %q, got: %q", want, row2Labels)
		}
	}

	for _, want := range []string{"8.0 R/G", "2", "6"} {
		if !strings.Contains(row2Values, want) {
			t.Errorf("row2 values missing %q, got: %q", want, row2Values)
		}
	}
}

// TestRenderH2HSkeletonComposesHorizontally is the loading-state
// counterpart of TestRenderH2HStatGridComposesHorizontally: the
// skeleton must lay out its label row the same way the loaded grid
// does, so the layout doesn't jump when data arrives.
func TestRenderH2HSkeletonComposesHorizontally(t *testing.T) {
	out := renderH2HSkeleton("AWY", "HOM", TeamColors{}, TeamColors{}, nil, 100)
	lines := strings.Split(out, "\n")

	var row1Labels, row2Labels string
	for _, l := range lines {
		if strings.Contains(l, "Runs") && strings.Contains(l, "Run diff") {
			row1Labels = l
		}
		if strings.Contains(l, "Avg total") && strings.Contains(l, "Largest margin") {
			row2Labels = l
		}
	}
	if row1Labels == "" {
		t.Fatalf("expected a line with Runs + Run diff + Avg score together, got: %q", lines)
	}
	if !strings.Contains(row1Labels, "Avg score") {
		t.Errorf("row1 labels missing Avg score, got: %q", row1Labels)
	}
	if row2Labels == "" {
		t.Fatalf("expected a line with Avg total + 1-run games + Largest margin together, got: %q", lines)
	}
	if !strings.Contains(row2Labels, "1-run games") {
		t.Errorf("row2 labels missing 1-run games, got: %q", row2Labels)
	}
}

func TestRenderH2HBodyShowsAggregates(t *testing.T) {
	p := &h2hPayload{
		loaded: true, games: 5,
		awayWins: 3, homeWins: 2,
		awayRunsTotal: 22, homeRunsTotal: 18,
		oneRunGames: 2, largestMargin: 6,
		lastMeeting: &h2hMeeting{date: "2026-05-30", awayRuns: 4, homeRuns: 3},
	}
	out := renderH2HBody(p, "HOU", "BAL", TeamColors{}, TeamColors{}, 100)
	for _, want := range []string{"HOU 3", "2 BAL", "(5G)", "22-18", "2026-05-30", "1-run games", "Largest margin", "6"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in body, got: %q", want, out)
		}
	}
}
