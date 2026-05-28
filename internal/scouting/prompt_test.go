package scouting

import (
	"strings"
	"testing"

	"github.com/pdavlin/go-playball/internal/api"
)

func TestRenderPrompt_IncludesSectionInstructions(t *testing.T) {
	system, _ := RenderPrompt(Context{})
	for _, header := range []string{"## The Edge", "## Pitching Edge", "## Bats to Watch"} {
		if !strings.Contains(system, header) {
			t.Errorf("system prompt missing required section header %q", header)
		}
	}
}

func TestRenderPrompt_OmitsStatsCleanlyWhenNil(t *testing.T) {
	ctx := Context{
		GamePk:        1,
		GameDateLocal: "Tue, May 27 at 7:10 PM PT",
		Venue:         "Dodger Stadium",
		Away: TeamCtx{
			Name:         "Los Angeles Dodgers",
			Abbreviation: "LAD",
			Record:       "30-20",
		},
		Home: TeamCtx{
			Name:         "San Francisco Giants",
			Abbreviation: "SF",
			Record:       "28-22",
		},
		Probables: [2]ProbableCtx{
			{Name: "Yoshinobu Yamamoto"},
			{Name: "Logan Webb"},
		},
	}
	_, user := RenderPrompt(ctx)
	if !strings.Contains(user, "Yoshinobu Yamamoto") {
		t.Error("user prompt missing away starter name")
	}
	if !strings.Contains(user, "stats unavailable") {
		t.Error("user prompt should mention stats unavailable when nil")
	}
	if strings.Contains(user, "ERA") {
		t.Error("user prompt should not contain ERA token when stats nil")
	}
}

func TestRenderPrompt_IncludesStatsWhenPresent(t *testing.T) {
	ctx := Context{
		Away: TeamCtx{
			Name:   "X",
			Record: "10-5",
			SeasonHit: &api.HittingLine{
				AVG: ".265", OBP: ".330", SLG: ".440", OPS: ".770",
				HomeRuns: 12, RBI: 50,
			},
		},
		Home: TeamCtx{Name: "Y", Record: "8-7"},
		Probables: [2]ProbableCtx{
			{Name: "A", SeasonLine: &api.PitchingLine{
				Wins: 4, Losses: 2, ERA: "3.10", WHIP: "1.05", K9: "10.1", IP: "62.0",
			}},
			{Name: "B"},
		},
	}
	_, user := RenderPrompt(ctx)
	if !strings.Contains(user, "ERA 3.10") {
		t.Errorf("expected ERA in prompt:\n%s", user)
	}
	if !strings.Contains(user, "AVG .265") {
		t.Errorf("expected AVG in prompt:\n%s", user)
	}
}

func TestRenderPrompt_LineupAbsentOmitsBlock(t *testing.T) {
	_, user := RenderPrompt(Context{
		Away: TeamCtx{Name: "X", Record: "1-0"},
		Home: TeamCtx{Name: "Y", Record: "0-1"},
	})
	if strings.Contains(user, "Lineups") {
		t.Errorf("Lineups block leaked into lineup-absent prompt:\n%s", user)
	}
}

func TestRenderPrompt_LineupPresentRendersBatters(t *testing.T) {
	ctx := Context{
		Away: TeamCtx{Name: "X", Record: "1-0"},
		Home: TeamCtx{Name: "Y", Record: "0-1"},
		Lineups: [2]LineupCtx{
			{Batters: []BatterCtx{
				{PlayerID: 1, Name: "Shohei Ohtani", Position: "DH", BattingOrder: 1,
					SeasonLine: &api.HittingLine{AVG: ".310", OBP: ".400", OPS: "1.000"}},
				{PlayerID: 2, Name: "Mookie Betts", Position: "RF", BattingOrder: 2},
			}},
			{Batters: []BatterCtx{
				{PlayerID: 10, Name: "Matt Chapman", Position: "3B", BattingOrder: 1,
					SeasonLine: &api.HittingLine{AVG: ".250", OBP: ".330", OPS: ".790"}},
			}},
		},
	}
	_, user := RenderPrompt(ctx)
	if !strings.Contains(user, "Lineups (top by batting order):") {
		t.Errorf("missing Lineups header:\n%s", user)
	}
	if !strings.Contains(user, "Shohei Ohtani") {
		t.Errorf("missing batter name:\n%s", user)
	}
	if !strings.Contains(user, "Mookie Betts") {
		t.Errorf("missing nil-stats batter:\n%s", user)
	}
	if !strings.Contains(user, "stats unavailable") {
		t.Errorf("nil SeasonLine should render stats unavailable:\n%s", user)
	}
	if !strings.Contains(user, "AVG .310") {
		t.Errorf("missing batter stat line:\n%s", user)
	}
}

func TestSystemPrompt_MentionsLineupGuidance(t *testing.T) {
	system, _ := RenderPrompt(Context{})
	if !strings.Contains(system, "Lineups") {
		t.Error("system prompt missing conditional Lineups guidance")
	}
}
