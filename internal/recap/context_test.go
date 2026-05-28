package recap

import (
	"context"
	"errors"
	"testing"

	"github.com/pdavlin/go-playball/internal/api"
)

func TestBuildContext_NotFinal(t *testing.T) {
	g := &api.Game{
		Status:   api.GameStatus{AbstractGameState: "Live"},
		LiveData: &api.LiveData{},
		GameData: &api.GameData{Status: api.GameStatus{AbstractGameState: "Live"}},
	}
	_, err := BuildContext(context.Background(), nil, g)
	if !errors.Is(err, ErrNotFinal) {
		t.Fatalf("want ErrNotFinal, got %v", err)
	}
}

func TestBuildContext_MissingLiveData(t *testing.T) {
	g := &api.Game{Status: api.GameStatus{AbstractGameState: "Final"}}
	// No client, no GameData/LiveData → cannot hydrate.
	_, err := BuildContext(context.Background(), nil, g)
	if !errors.Is(err, ErrIncompletePayload) {
		t.Fatalf("want ErrIncompletePayload, got %v", err)
	}
}

func TestBuildContext_FinalWithZeroScoringAndNoDecisions(t *testing.T) {
	g := &api.Game{
		Status:   api.GameStatus{AbstractGameState: "Final"},
		GameData: &api.GameData{Status: api.GameStatus{AbstractGameState: "Final"}},
		LiveData: &api.LiveData{},
	}
	_, err := BuildContext(context.Background(), nil, g)
	if !errors.Is(err, ErrIncompletePayload) {
		t.Fatalf("want ErrIncompletePayload, got %v", err)
	}
}

func TestBuildContext_HappyPath(t *testing.T) {
	g := buildFinalGameFixture()
	ctx, err := BuildContext(context.Background(), nil, g)
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}

	if ctx.Away.Runs != 5 || ctx.Home.Runs != 3 {
		t.Errorf("score mismatch: away=%d home=%d", ctx.Away.Runs, ctx.Home.Runs)
	}
	if ctx.Decisions.Winner == nil || ctx.Decisions.Winner.Name != "Jane Winner" {
		t.Errorf("winner not extracted: %+v", ctx.Decisions.Winner)
	}
	if ctx.Decisions.Save == nil || ctx.Decisions.Save.Name != "Sam Saver" {
		t.Errorf("save not extracted: %+v", ctx.Decisions.Save)
	}

	awayHitters := ctx.Standouts[0].Hitters
	if len(awayHitters) == 0 || awayHitters[0].Name != "Three Hits" {
		t.Errorf("top away hitter wrong: %+v", awayHitters)
	}

	awayPitchers := ctx.Standouts[0].Pitchers
	var winnerLine *PitcherLine
	for i := range awayPitchers {
		if awayPitchers[i].Decision == "W" {
			winnerLine = &awayPitchers[i]
		}
	}
	if winnerLine == nil {
		t.Fatalf("winning pitcher not tagged in away standouts: %+v", awayPitchers)
	}
	if winnerLine.Name != "Jane Winner" {
		t.Errorf("winning pitcher name wrong: %s", winnerLine.Name)
	}

	if len(ctx.Scoring) == 0 {
		t.Errorf("expected scoring plays extracted")
	}
}

func TestTopHitters_PrefersHighHits(t *testing.T) {
	players := map[string]api.BoxscorePlayer{
		"ID1": {Person: api.Player{ID: 1, FullName: "One Hit"},
			Stats: api.PlayerStats{Batting: &api.BattingStats{AtBats: 4, Hits: 1}}},
		"ID2": {Person: api.Player{ID: 2, FullName: "Three Hits"},
			Stats: api.PlayerStats{Batting: &api.BattingStats{AtBats: 4, Hits: 3, RBI: 2}}},
		"ID3": {Person: api.Player{ID: 3, FullName: "Two Hits"},
			Stats: api.PlayerStats{Batting: &api.BattingStats{AtBats: 4, Hits: 2}}},
	}
	got := topHitters(players)
	if len(got) != 3 || got[0].Name != "Three Hits" || got[1].Name != "Two Hits" {
		t.Errorf("unexpected order: %+v", got)
	}
}

func TestParseInningsToOuts(t *testing.T) {
	cases := map[string]int{
		"7.0": 21,
		"7.1": 22,
		"7.2": 23,
		"7":   21,
		"0.2": 2,
		"":    0,
	}
	for in, want := range cases {
		if got := parseInningsToOuts(in); got != want {
			t.Errorf("parseInningsToOuts(%q) = %d, want %d", in, got, want)
		}
	}
}

func buildFinalGameFixture() *api.Game {
	awayPlayers := map[string]api.BoxscorePlayer{
		"ID101": {
			Person: api.Player{ID: 101, FullName: "Three Hits"},
			Stats:  api.PlayerStats{Batting: &api.BattingStats{AtBats: 4, Hits: 3, RBI: 2}},
		},
		"ID102": {
			Person: api.Player{ID: 102, FullName: "Quiet Bat"},
			Stats:  api.PlayerStats{Batting: &api.BattingStats{AtBats: 4, Hits: 0}},
		},
		"ID201": {
			Person: api.Player{ID: 201, FullName: "Jane Winner"},
			Stats:  api.PlayerStats{Pitching: &api.PitchingStats{InningsPitched: "7.0", StrikeOuts: 8, BaseOnBalls: 1, EarnedRuns: 1}},
		},
	}
	homePlayers := map[string]api.BoxscorePlayer{
		"ID301": {
			Person: api.Player{ID: 301, FullName: "Sam Saver"},
			Stats:  api.PlayerStats{Pitching: &api.PitchingStats{InningsPitched: "1.0", StrikeOuts: 1, BaseOnBalls: 0, EarnedRuns: 0}},
		},
		"ID302": {
			Person: api.Player{ID: 302, FullName: "Bob Loser"},
			Stats:  api.PlayerStats{Pitching: &api.PitchingStats{InningsPitched: "5.0", StrikeOuts: 4, BaseOnBalls: 3, EarnedRuns: 4}},
		},
	}

	one := 1
	zero := 0
	innings := []api.Inning{
		{Num: 1, Away: api.InningScore{Runs: &one}, Home: api.InningScore{Runs: &zero}},
		{Num: 2, Away: api.InningScore{Runs: &zero}, Home: api.InningScore{Runs: &zero}},
	}

	return &api.Game{
		ID:     12345,
		Status: api.GameStatus{AbstractGameState: "Final"},
		Teams: api.Teams{
			Away: api.TeamInfo{Team: api.Team{ID: 1, Name: "Away Town", Abbreviation: "AWY"}},
			Home: api.TeamInfo{Team: api.Team{ID: 2, Name: "Home Field", Abbreviation: "HOM"}},
		},
		GameData: &api.GameData{
			Status: api.GameStatus{AbstractGameState: "Final"},
			Venue:  api.Venue{Name: "Test Park"},
		},
		LiveData: &api.LiveData{
			Decisions: api.Decisions{
				Winner: &api.DecisionPitcher{ID: 201, FullName: "Jane Winner"},
				Loser:  &api.DecisionPitcher{ID: 302, FullName: "Bob Loser"},
				Save:   &api.DecisionPitcher{ID: 301, FullName: "Sam Saver"},
			},
			Linescore: api.Linescore{
				Innings: innings,
				Teams: api.LinescoreTeams{
					Away: api.LinescoreTeam{Runs: 5, Hits: 9, Errors: 0},
					Home: api.LinescoreTeam{Runs: 3, Hits: 7, Errors: 1},
				},
			},
			Plays: api.Plays{
				AllPlays: []api.Play{
					{
						About:  api.About{Inning: 1, HalfInning: "top", IsScoringPlay: true},
						Result: api.PlayResult{Description: "Three Hits doubles on a line drive to right.", AwayScore: 1, HomeScore: 0},
					},
				},
			},
			Boxscore: api.Boxscore{
				Teams: api.BoxscoreTeams{
					Away: api.BoxscoreTeam{Players: awayPlayers},
					Home: api.BoxscoreTeam{Players: homePlayers},
				},
			},
		},
	}
}
