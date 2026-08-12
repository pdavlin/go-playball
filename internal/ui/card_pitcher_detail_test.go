package ui

import (
	"errors"
	"strings"
	"testing"

	"github.com/pdavlin/go-playball/internal/api"
)

func TestRenderArsenalTableEmpty(t *testing.T) {
	out := renderArsenalTable(nil)
	if !strings.Contains(out, "Arsenal unavailable") {
		t.Errorf("expected 'Arsenal unavailable' fallback, got: %q", out)
	}
}

func TestRenderArsenalTableMissingVelocity(t *testing.T) {
	rows := []api.ArsenalPitch{
		{Code: "FF", Description: "Four-seam FB", UsagePct: 50.0, AvgVelocity: 0},
		{Code: "SL", Description: "Slider", UsagePct: 50.0, AvgVelocity: 87.4},
	}
	out := renderArsenalTable(rows)
	if !strings.Contains(out, "Four-seam FB") {
		t.Errorf("expected pitch name in output, got: %q", out)
	}
	// The velocity cell for the no-velocity row should render "-".
	if !strings.Contains(out, "-") {
		t.Errorf("expected '-' for missing velocity, got: %q", out)
	}
	if !strings.Contains(out, "87.4") {
		t.Errorf("expected '87.4' for present velocity, got: %q", out)
	}
}

func TestRenderRecentStartsEmpty(t *testing.T) {
	out := renderRecentStartsTable(nil)
	if !strings.Contains(out, "No prior starts this season") {
		t.Errorf("expected empty fallback, got: %q", out)
	}
}

func TestRenderRecentStartsFormatting(t *testing.T) {
	rows := []api.GameLogStart{
		{
			Date: "2026-04-12", OpponentName: "Baltimore Orioles",
			InningsPitched: "6.0", EarnedRuns: 2, Strikeouts: 7,
			Walks: 1, HomeRuns: 1, Pitches: 92, IsStart: true,
		},
	}
	out := renderRecentStartsTable(rows)
	if !strings.Contains(out, "2026-04-12") {
		t.Errorf("expected date in output, got: %q", out)
	}
	if !strings.Contains(out, "BAL") {
		t.Errorf("expected opponent abbreviation BAL, got: %q", out)
	}
	if !strings.Contains(out, "6.0") {
		t.Errorf("expected IP value, got: %q", out)
	}
	if !strings.Contains(out, "92") {
		t.Errorf("expected pitch count, got: %q", out)
	}
}

func TestRenderPitcherColumnTBD(t *testing.T) {
	colors := TeamColors{}
	out := renderPitcherColumn(api.ProbablePitcher{ID: 0}, colors, nil, nil, nil)
	if !strings.Contains(out, "Probable pitcher TBD") {
		t.Errorf("expected TBD fallback, got: %q", out)
	}
}

func TestRenderPitcherColumnFetchError(t *testing.T) {
	colors := TeamColors{}
	prob := api.ProbablePitcher{ID: 543037, FullName: "Gerrit Cole"}
	out := renderPitcherColumn(prob, colors, nil, nil, errors.New("net down"))
	if !strings.Contains(out, "Data unavailable") {
		t.Errorf("expected error fallback, got: %q", out)
	}
	if !strings.Contains(out, "Gerrit Cole") {
		t.Errorf("expected pitcher name even on error, got: %q", out)
	}
}

func TestRenderPitcherDetailCardSkeletonOnLoad(t *testing.T) {
	game := &api.Game{
		GameData: &api.GameData{
			Teams: api.GameTeams{
				Away: api.FullTeamInfo{Name: "Houston Astros"},
				Home: api.FullTeamInfo{Name: "Baltimore Orioles"},
			},
			ProbablePitchers: api.ProbablePitchers{
				Away: api.ProbablePitcher{ID: 1, FullName: "Pitcher A"},
				Home: api.ProbablePitcher{ID: 2, FullName: "Pitcher B"},
			},
		},
	}
	out := renderPitcherDetailCard(game, nil, nil, 120)
	// The skeleton path should render the table headers + pitcher
	// names, NOT the old "Loading pitchers..." placeholder.
	if strings.Contains(out, "Loading pitchers") {
		t.Errorf("expected skeleton tables, got fallback string: %q", out)
	}
	for _, want := range []string{"Pitcher A", "Pitcher B", "Arsenal", "Recent starts", "Pitch", "Date", "Opp"} {
		if !strings.Contains(out, want) {
			t.Errorf("skeleton output missing %q, got: %q", want, out)
		}
	}
}

func TestRenderPitcherDetailCardLayoutSwitch(t *testing.T) {
	game := &api.Game{
		GameData: &api.GameData{
			Teams: api.GameTeams{
				Away: api.FullTeamInfo{Name: "Houston Astros"},
				Home: api.FullTeamInfo{Name: "Baltimore Orioles"},
			},
			ProbablePitchers: api.ProbablePitchers{
				Away: api.ProbablePitcher{ID: 1, FullName: "Pitcher A"},
				Home: api.ProbablePitcher{ID: 2, FullName: "Pitcher B"},
			},
		},
	}
	data := &pregameGameData{
		pitcherDetail: &pitcherDetailPayload{
			awayArsenal: []api.ArsenalPitch{
				{Code: "FF", Description: "Four-seam FB", UsagePct: 60, AvgVelocity: 95},
			},
			homeArsenal: []api.ArsenalPitch{
				{Code: "SL", Description: "Slider", UsagePct: 40, AvgVelocity: 86},
			},
			loaded: true,
		},
	}

	wide := renderPitcherDetailCard(game, data, nil, 120)
	narrow := renderPitcherDetailCard(game, data, nil, 80)

	if strings.Count(wide, "Pitcher A") == 0 || strings.Count(wide, "Pitcher B") == 0 {
		t.Fatalf("wide layout missing pitchers, got: %q", wide)
	}
	if strings.Count(narrow, "Pitcher A") == 0 || strings.Count(narrow, "Pitcher B") == 0 {
		t.Fatalf("narrow layout missing pitchers, got: %q", narrow)
	}

	// Stacked (narrow) layout should put Pitcher B strictly below
	// Pitcher A on a different line; side-by-side should have them on
	// the same line for the header row.
	wideLines := strings.Split(wide, "\n")
	sameLine := false
	for _, ln := range wideLines {
		if strings.Contains(ln, "Pitcher A") && strings.Contains(ln, "Pitcher B") {
			sameLine = true
			break
		}
	}
	if !sameLine {
		t.Error("wide layout should render both pitcher names on the same line")
	}
	narrowLines := strings.Split(narrow, "\n")
	for _, ln := range narrowLines {
		if strings.Contains(ln, "Pitcher A") && strings.Contains(ln, "Pitcher B") {
			t.Error("narrow layout should not render both pitcher names on the same line")
		}
	}
}

func TestRenderRecentStartsTruncatedByCaller(t *testing.T) {
	// The renderer itself does not truncate; the API function caps at
	// limit. Verify the renderer prints exactly the rows it's given.
	rows := make([]api.GameLogStart, 7)
	for i := range rows {
		rows[i] = api.GameLogStart{Date: "2026-05-01", OpponentName: "Test"}
	}
	out := renderRecentStartsTable(rows)
	if got := strings.Count(out, "2026-05-01"); got != 7 {
		t.Errorf("renderer should render every row it's given; got %d rows", got)
	}
}
