package scouting

import (
	"strings"
	"testing"

	"github.com/pdavlin/go-playball/internal/api"
)

// --- in-progress system prompt ---

func TestInProgressSystemPrompt_PresentTenseAndInProgress(t *testing.T) {
	system, _ := RenderPrompt(Context{State: StateInProgress})
	if !strings.Contains(system, "IN PROGRESS") {
		t.Error("in-progress system prompt should mark the game as IN PROGRESS")
	}
	if !strings.Contains(system, "present tense") {
		t.Error("in-progress system prompt should instruct present tense")
	}
	// The pregame headers must not leak into the in-progress tense.
	if strings.Contains(system, "## The Edge") {
		t.Error("in-progress prompt should not carry the pregame ## The Edge header")
	}
	for _, h := range []string{"## The Duel", "## Hitter Watch", "## What to Watch"} {
		if !strings.Contains(system, h) {
			t.Errorf("in-progress system prompt missing header %q", h)
		}
	}
}

func TestInProgressSystemPrompt_ForbidsPlayByPlay(t *testing.T) {
	system, _ := RenderPrompt(Context{State: StateInProgress})
	if !strings.Contains(system, "Never write a play-by-play sentence") {
		t.Error("in-progress system prompt should forbid play-by-play sentences")
	}
	if !strings.Contains(system, "scoring event that is not in the facts") {
		t.Error("in-progress system prompt should forbid inventing scoring events")
	}
}

func TestInProgressSystemPrompt_DoNotRestateScore(t *testing.T) {
	system, _ := RenderPrompt(Context{State: StateInProgress})
	if !strings.Contains(system, "do not restate the score") {
		t.Error("in-progress system prompt should tell the model not to restate the score")
	}
	if !strings.Contains(system, "scoreless duel") {
		t.Error("in-progress system prompt should handle the no-scoring-yet case")
	}
}

func TestInProgressSystemPrompt_SharesGroundingAndSavantRules(t *testing.T) {
	system, _ := RenderPrompt(Context{State: StateInProgress})
	if !strings.Contains(system, "must appear verbatim") {
		t.Error("in-progress prompt should carry the verbatim-number grounding rule")
	}
	if !strings.Contains(system, "savant pct rank") {
		t.Error("in-progress prompt should reuse the Savant xStats guidance")
	}
	if !strings.Contains(system, "make the GAP the story") {
		t.Error("in-progress prompt should reuse the expected-vs-actual gap guidance")
	}
}

// --- in-progress user prompt (live block) ---

func inProgressCtx(live *LiveCtx) Context {
	return Context{
		State: StateInProgress,
		Away:  TeamCtx{Name: "Los Angeles Dodgers", Abbreviation: "LAD", Record: "60-40"},
		Home:  TeamCtx{Name: "San Francisco Giants", Abbreviation: "SF", Record: "55-45"},
		Probables: [2]ProbableCtx{
			{Name: "Yoshinobu Yamamoto", SeasonLine: &api.PitchingLine{Wins: 10, Losses: 4, ERA: "2.80", WHIP: "1.00", K9: "10.0", IP: "120.0"}},
			{Name: "Logan Webb", SeasonLine: &api.PitchingLine{Wins: 9, Losses: 8, ERA: "3.40", WHIP: "1.15", K9: "8.5", IP: "150.0"}},
		},
		Live: live,
	}
}

func TestRenderPrompt_InProgressLiveBlock(t *testing.T) {
	live := &LiveCtx{
		Inning: 6, InningState: "Bottom", InningOrdinal: "6th", Outs: 2,
		AwayRuns: 3, HomeRuns: 2,
		OnFirst: true, OnSecond: true,
		Starters: [2]LiveStarterLine{
			{Name: "Yoshinobu Yamamoto", IP: "5.0", H: 4, R: 2, ER: 2, K: 6, Pitches: 88, Present: true},
			{Name: "Logan Webb", IP: "6.0", H: 5, R: 3, ER: 3, K: 4, Present: true},
		},
		Standouts: [2][]LiveHitterLine{
			{{Name: "Mookie Betts", AB: 3, H: 2, HR: 1, RBI: 2}},
			{{Name: "Matt Chapman", AB: 4, H: 2, RBI: 1}},
		},
	}
	_, user := RenderPrompt(inProgressCtx(live))

	if !strings.Contains(user, "Live game state") {
		t.Errorf("missing live game state block:\n%s", user)
	}
	if !strings.Contains(user, "Situation: bottom of the 6th, 2 out") {
		t.Errorf("inning/outs line missing or malformed:\n%s", user)
	}
	if !strings.Contains(user, "Score: Away 3, Home 2") {
		t.Errorf("score line missing or malformed:\n%s", user)
	}
	if !strings.Contains(user, "Bases: runners on first and second") {
		t.Errorf("bases line missing or malformed:\n%s", user)
	}
	if !strings.Contains(user, "Away starter Yoshinobu Yamamoto so far: 5.0 IP, 4 H, 2 R, 2 ER, 6 K, 88 pitches") {
		t.Errorf("away starter so-far line missing or malformed:\n%s", user)
	}
	if !strings.Contains(user, "Home starter Logan Webb so far: 6.0 IP, 5 H, 3 R, 3 ER, 4 K") {
		t.Errorf("home starter so-far line missing or malformed:\n%s", user)
	}
	if strings.Contains(user, "Logan Webb so far: 6.0 IP, 5 H, 3 R, 3 ER, 4 K, ") {
		t.Errorf("pitch count should be omitted when zero:\n%s", user)
	}
	if !strings.Contains(user, "Mookie Betts: 2-for-3, 1 HR, 2 RBI") {
		t.Errorf("away standout line missing or malformed:\n%s", user)
	}
	// The season spine must still be present so the pitching duel has ERAs.
	if !strings.Contains(user, "ERA 2.80") {
		t.Errorf("season pitching line should still appear in the in-progress spine:\n%s", user)
	}
}

func TestRenderPrompt_InProgressScoreless(t *testing.T) {
	live := &LiveCtx{
		Inning: 3, InningState: "Top", InningOrdinal: "3rd", Outs: 1,
	}
	_, user := RenderPrompt(inProgressCtx(live))
	if !strings.Contains(user, "Score: scoreless, no runs yet") {
		t.Errorf("scoreless case should read naturally:\n%s", user)
	}
	if !strings.Contains(user, "Bases: empty") {
		t.Errorf("empty bases should render 'empty':\n%s", user)
	}
}

func TestRenderPrompt_PregameHasNoLiveBlock(t *testing.T) {
	_, user := RenderPrompt(Context{
		Away: TeamCtx{Name: "X", Record: "1-0"},
		Home: TeamCtx{Name: "Y", Record: "0-1"},
	})
	if strings.Contains(user, "Live game state") {
		t.Errorf("pregame prompt should not carry a live-state block:\n%s", user)
	}
}

// --- buildLiveContext against a synthetic live game ---

func TestBuildLiveContext_PopulatesLiveFacts(t *testing.T) {
	runs := 2
	g := &api.Game{
		LiveData: &api.LiveData{
			Linescore: api.Linescore{
				CurrentInning:        7,
				CurrentInningOrdinal: "7th",
				InningState:          "Top",
				Outs:                 1,
				Teams: api.LinescoreTeams{
					Away: api.LinescoreTeam{Runs: 4},
					Home: api.LinescoreTeam{Runs: runs},
				},
				Offense: api.Offense{
					First: &api.Player{ID: 99, FullName: "Runner"},
					Third: &api.Player{ID: 98, FullName: "Runner3"},
				},
			},
			Boxscore: api.Boxscore{
				Teams: api.BoxscoreTeams{
					Away: api.BoxscoreTeam{
						Pitchers: []int{10},
						Players: map[string]api.BoxscorePlayer{
							"ID10": {
								Person: api.Player{ID: 10, FullName: "Away Starter"},
								Stats: api.PlayerStats{Pitching: &api.PitchingStats{
									InningsPitched: "6.0", Hits: 3, Runs: 1, EarnedRuns: 1, StrikeOuts: 7, PitchesThrown: 92,
								}},
							},
							"ID20": {
								Person: api.Player{ID: 20, FullName: "Big Bat"},
								Stats: api.PlayerStats{Batting: &api.BattingStats{
									AtBats: 4, Hits: 3, HomeRuns: 1, RBI: 3,
								}},
							},
							"ID21": {
								Person: api.Player{ID: 21, FullName: "Hitless Guy"},
								Stats: api.PlayerStats{Batting: &api.BattingStats{
									AtBats: 4, Hits: 0,
								}},
							},
						},
					},
					Home: api.BoxscoreTeam{
						Pitchers: []int{30},
						Players: map[string]api.BoxscorePlayer{
							"ID30": {
								Person: api.Player{ID: 30, FullName: "Home Starter"},
								Stats: api.PlayerStats{Pitching: &api.PitchingStats{
									InningsPitched: "5.2", Hits: 6, Runs: 4, EarnedRuns: 4, StrikeOuts: 3,
								}},
							},
						},
					},
				},
			},
		},
	}

	live := buildLiveContext(g)
	if live == nil {
		t.Fatal("buildLiveContext returned nil for a live game")
	}
	if live.Inning != 7 || live.InningOrdinal != "7th" || live.InningState != "Top" || live.Outs != 1 {
		t.Errorf("inning/outs not populated: %+v", live)
	}
	if live.AwayRuns != 4 || live.HomeRuns != 2 {
		t.Errorf("scores not populated: away=%d home=%d", live.AwayRuns, live.HomeRuns)
	}
	if !live.OnFirst || live.OnSecond || !live.OnThird {
		t.Errorf("base state wrong: first=%v second=%v third=%v", live.OnFirst, live.OnSecond, live.OnThird)
	}
	if !live.Starters[0].Present || live.Starters[0].Name != "Away Starter" ||
		live.Starters[0].IP != "6.0" || live.Starters[0].K != 7 || live.Starters[0].Pitches != 92 {
		t.Errorf("away starter line wrong: %+v", live.Starters[0])
	}
	if live.Starters[1].Name != "Home Starter" || live.Starters[1].ER != 4 {
		t.Errorf("home starter line wrong: %+v", live.Starters[1])
	}
	// Only the batter with a hit is a standout; the hitless one is dropped.
	if len(live.Standouts[0]) != 1 || live.Standouts[0][0].Name != "Big Bat" {
		t.Errorf("away standouts wrong: %+v", live.Standouts[0])
	}
	if len(live.Standouts[1]) != 0 {
		t.Errorf("home should have no standouts (no batters with a hit): %+v", live.Standouts[1])
	}
}

func TestBuildLiveContext_NilWhenNoLiveData(t *testing.T) {
	if got := buildLiveContext(&api.Game{}); got != nil {
		t.Errorf("expected nil live context when LiveData is absent, got %+v", got)
	}
}

func TestIsInProgress(t *testing.T) {
	live := &api.Game{Status: api.GameStatus{AbstractGameState: "Live"}}
	if !isInProgress(live) {
		t.Error("schedule-side Live status should be detected as in progress")
	}
	preview := &api.Game{Status: api.GameStatus{AbstractGameState: "Preview"}}
	if isInProgress(preview) {
		t.Error("Preview game should not be in progress")
	}
	// gameData status wins over the schedule status when present.
	hydrated := &api.Game{
		Status:   api.GameStatus{AbstractGameState: "Preview"},
		GameData: &api.GameData{Status: api.GameStatus{AbstractGameState: "Live"}},
	}
	if !isInProgress(hydrated) {
		t.Error("hydrated gameData Live status should win over schedule Preview")
	}
}
