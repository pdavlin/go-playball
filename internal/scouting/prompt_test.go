package scouting

import (
	"strings"
	"testing"

	"github.com/pdavlin/go-playball/internal/api"
	"github.com/pdavlin/go-playball/internal/savant"
)

// ptr is a test helper returning a pointer to an int percentile.
func ptr(v int) *int { return &v }

func TestRenderPrompt_IncludesSectionInstructions(t *testing.T) {
	system, _ := RenderPrompt(Context{})
	for _, header := range []string{"## The Edge", "## Pitching Edge", "## Bats to Watch"} {
		if !strings.Contains(system, header) {
			t.Errorf("system prompt missing required section header %q", header)
		}
	}
}

func TestSystemPrompt_FramesFactsAsVerifiedSpine(t *testing.T) {
	system, _ := RenderPrompt(Context{})
	if !strings.Contains(system, "verified spine") {
		t.Error("system prompt should frame the supplied facts as the verified spine")
	}
	if !strings.Contains(system, "must appear verbatim") {
		t.Error("system prompt missing the verbatim-number grounding rule")
	}
	if !strings.Contains(system, "Do not compute a new number") {
		t.Error("system prompt should forbid computing new numbers or gaps")
	}
}

func TestSystemPrompt_ForbidsInventedCareerRecord(t *testing.T) {
	system, _ := RenderPrompt(Context{})
	if !strings.Contains(system, "career-vs-opponent") {
		t.Error("system prompt should forbid inventing a career-vs-opponent record")
	}
	if !strings.Contains(system, `"[Pitcher] is N-M vs [Team]"`) {
		t.Error("system prompt should name the forbidden N-M-vs-Team phrasing")
	}
}

func TestSystemPrompt_EdgeIsAnalyticalNotPrediction(t *testing.T) {
	system, _ := RenderPrompt(Context{})
	if !strings.Contains(system, "never a win prediction") {
		t.Error("The Edge should be stated as an analytical edge, not a win prediction")
	}
}

func TestSystemPrompt_VoiceRules(t *testing.T) {
	system, _ := RenderPrompt(Context{})
	if !strings.Contains(system, "sentence case") {
		t.Error("system prompt should require sentence-case titles and prose")
	}
	if !strings.Contains(system, "Avoid hype words") {
		t.Error("system prompt should carry the hype-word avoidance rule")
	}
	// The appended-section examples must themselves model sentence case.
	if !strings.Contains(system, "## Hot bats, ## Trending, ## X-factor") {
		t.Error("appended-section examples should be sentence case (## Hot bats), not Title Case")
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

func TestRenderPrompt_ConditionsLine(t *testing.T) {
	ctx := Context{
		Weather:  api.Weather{Condition: "Partly Cloudy", Temp: "80", Wind: "4 mph, Out To LF"},
		DayNight: "night",
	}
	_, user := RenderPrompt(ctx)
	if !strings.Contains(user, "Conditions: 80°F, Partly Cloudy, wind 4 mph, Out To LF, night game") {
		t.Errorf("conditions line missing or malformed:\n%s", user)
	}
}

func TestRenderPrompt_ConditionsOmittedWhenEmpty(t *testing.T) {
	_, user := RenderPrompt(Context{})
	if strings.Contains(user, "Conditions:") {
		t.Errorf("Conditions line leaked into prompt with no weather:\n%s", user)
	}
}

func TestRenderPrompt_TeamForm(t *testing.T) {
	ctx := Context{
		Away: TeamCtx{Name: "X", Record: "50-40", Streak: "W3", LastTen: "7-3",
			DivisionRank: "2", GamesBack: "1.5"},
		Home: TeamCtx{Name: "Y", Record: "40-50"},
	}
	_, user := RenderPrompt(ctx)
	if !strings.Contains(user, "form: streak W3, last ten 7-3, division rank 2 (1.5 GB)") {
		t.Errorf("team form line missing or malformed:\n%s", user)
	}
	if strings.Count(user, "form:") != 1 {
		t.Errorf("form line should be omitted for the team without standings data:\n%s", user)
	}
}

func TestRenderPrompt_TeamFormLeaderOmitsGB(t *testing.T) {
	ctx := Context{
		Away: TeamCtx{Name: "X", Record: "60-30", DivisionRank: "1", GamesBack: "-"},
	}
	_, user := RenderPrompt(ctx)
	if !strings.Contains(user, "form: division rank 1\n") {
		t.Errorf("leader form should omit GB when gamesBack is \"-\":\n%s", user)
	}
}

func TestRenderPrompt_ProbableHandArsenalAndStarts(t *testing.T) {
	ctx := Context{
		Probables: [2]ProbableCtx{
			{
				Name:        "Kodai Senga",
				HandsThrows: "RHP",
				Arsenal: []api.ArsenalPitch{
					{Description: "Splitter", UsagePct: 25, AvgVelocity: 84.2},
					{Description: "Four-Seam Fastball", UsagePct: 40, AvgVelocity: 96.1},
				},
				RecentStarts: []api.GameLogStart{
					{Date: "2026-07-08", OpponentName: "Atlanta Braves",
						InningsPitched: "6.0", EarnedRuns: 2, Strikeouts: 8, Walks: 1},
				},
			},
			{Name: "TBD placeholder"},
		},
	}
	_, user := RenderPrompt(ctx)
	if !strings.Contains(user, "Kodai Senga (RHP)") {
		t.Errorf("missing handedness label:\n%s", user)
	}
	if !strings.Contains(user, "arsenal: Four-Seam Fastball 40% (96.1 mph), Splitter 25% (84.2 mph)") {
		t.Errorf("arsenal missing or not sorted by usage:\n%s", user)
	}
	if !strings.Contains(user, "2026-07-08 vs Atlanta Braves: 6.0 IP, 2 ER, 8 K, 1 BB, 0 HR") {
		t.Errorf("recent start line missing:\n%s", user)
	}
}

func TestRenderPrompt_BatterSideAndRecentForm(t *testing.T) {
	ctx := Context{
		Lineups: [2]LineupCtx{
			{Batters: []BatterCtx{
				{PlayerID: 1, Name: "Juan Soto", Position: "RF", BatSide: "L", BattingOrder: 1,
					SeasonLine: &api.HittingLine{AVG: ".280", OBP: ".420", OPS: ".950"},
					Recent:     &api.BatterWindowStats{AVG: ".333", OPS: "1.100", HomeRuns: 3, GamesPlayed: 7}},
			}},
			{Batters: []BatterCtx{
				{PlayerID: 2, Name: "Pete Alonso", Position: "1B", BattingOrder: 1},
			}},
		},
	}
	_, user := RenderPrompt(ctx)
	if !strings.Contains(user, "Juan Soto RF (L): AVG .280 / OBP .420 / OPS .950; last 7 games: AVG .333, OPS 1.100, 3 HR") {
		t.Errorf("batter line missing side or recent form:\n%s", user)
	}
	if !strings.Contains(user, "Pete Alonso 1B: stats unavailable") {
		t.Errorf("no-side no-stats batter line malformed:\n%s", user)
	}
}

func TestSystemPrompt_MentionsLineupGuidance(t *testing.T) {
	system, _ := RenderPrompt(Context{})
	if !strings.Contains(system, "Lineups") {
		t.Error("system prompt missing conditional Lineups guidance")
	}
}

func TestSystemPrompt_AllowsSavantExpectedStats(t *testing.T) {
	system, _ := RenderPrompt(Context{})
	// The blanket prohibition on expected/percentile stats must be gone now
	// that Savant supplies them.
	if strings.Contains(system, "or expected/percentile stats") {
		t.Error("system prompt still carries the blanket expected/percentile prohibition")
	}
	// The expected-vs-actual guidance must be present and conditional on a
	// supplied savant line.
	if !strings.Contains(system, "savant pct rank") {
		t.Error("system prompt should key expected-stat use to a supplied savant line")
	}
	if !strings.Contains(system, "make the GAP the story") {
		t.Error("system prompt should instruct making the expected-vs-actual gap the story")
	}
	// The grounding rule against inventing a computed gap must survive.
	if !strings.Contains(system, "Never state a numeric gap the facts do not contain") {
		t.Error("system prompt should still forbid inventing a computed numeric gap")
	}
}

func TestSystemPrompt_StillForbidsInventedExpectedStatsWhenAbsent(t *testing.T) {
	system, _ := RenderPrompt(Context{})
	if !strings.Contains(system, "do not cite or estimate them") {
		t.Error("system prompt should still forbid expected stats where no savant line is supplied")
	}
}

func TestRenderPrompt_BatterSavantLine(t *testing.T) {
	ctx := Context{
		Lineups: [2]LineupCtx{
			{Batters: []BatterCtx{
				{PlayerID: 1, Name: "Colt Keith", Position: "3B", BattingOrder: 1,
					SeasonLine: &api.HittingLine{AVG: ".270", OBP: ".330", OPS: ".780"},
					XStats: &savant.Percentiles{
						XWOBAPct: ptr(59), WOBAPct: ptr(46), XBAPct: ptr(81),
						HardHitPct: ptr(48), KPct: ptr(66), ChasePct: ptr(81),
						SprintSpeedPct: ptr(70),
					}},
			}},
			{Batters: []BatterCtx{
				{PlayerID: 2, Name: "No Savant Guy", Position: "2B", BattingOrder: 1},
			}},
		},
	}
	_, user := RenderPrompt(ctx)
	if !strings.Contains(user, "savant pct rank: xwOBA 59, wOBA 46, xBA 81, hard-hit 48, K 66, chase 81, sprint 70") {
		t.Errorf("batter savant line missing or malformed:\n%s", user)
	}
	// The batter with no XStats must not emit a savant line.
	if strings.Count(user, "savant pct rank") != 1 {
		t.Errorf("expected exactly one savant line, got:\n%s", user)
	}
}

func TestRenderPrompt_BatterSavantNilFieldsDropped(t *testing.T) {
	ctx := Context{
		Lineups: [2]LineupCtx{
			{Batters: []BatterCtx{
				{PlayerID: 1, Name: "Low Sample", Position: "SS", BattingOrder: 1,
					XStats: &savant.Percentiles{SprintSpeedPct: ptr(81)}},
			}},
			{Batters: []BatterCtx{{PlayerID: 2, Name: "Filler", BattingOrder: 1}}},
		},
	}
	_, user := RenderPrompt(ctx)
	if !strings.Contains(user, "savant pct rank: sprint 81") {
		t.Errorf("low-sample batter should show only the populated rank:\n%s", user)
	}
	if strings.Contains(user, "xwOBA") {
		t.Errorf("nil percentile fields should be dropped, not rendered:\n%s", user)
	}
}

func TestRenderPrompt_PitcherSavantLine(t *testing.T) {
	ctx := Context{
		Probables: [2]ProbableCtx{
			{
				Name:        "Ace Starter",
				HandsThrows: "RHP",
				SeasonLine:  &api.PitchingLine{Wins: 8, Losses: 4, ERA: "3.80", WHIP: "1.10", K9: "9.0", IP: "120.0"},
				XStats: &savant.Percentiles{
					XERAPct: ptr(78), KPct: ptr(70), WhiffPct: ptr(72),
					ChasePct: ptr(60), FastballVeloPct: ptr(85), BarrelPct: ptr(55),
				},
			},
			{Name: "No Savant"},
		},
	}
	_, user := RenderPrompt(ctx)
	if !strings.Contains(user, "savant pct rank: xERA 78, K 70, whiff 72, chase 60, fastball velo 85, barrel 55") {
		t.Errorf("pitcher savant line missing or malformed:\n%s", user)
	}
	if strings.Count(user, "savant pct rank") != 1 {
		t.Errorf("only the starter with XStats should emit a savant line:\n%s", user)
	}
}
