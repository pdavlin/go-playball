package ui

import (
	"testing"

	"github.com/pdavlin/go-playball/internal/api"
)

func gameWithState(state string) *api.Game {
	return &api.Game{Status: api.GameStatus{AbstractGameState: state}}
}

func TestReportKindFor(t *testing.T) {
	cases := []struct {
		name      string
		enabled   bool
		game      *api.Game
		wantKind  reportKind
		wantLabel string
		wantOK    bool
	}{
		{"preview routes to scouting", true, gameWithState("Preview"), reportKindScouting, "scouting", true},
		{"live routes to scouting", true, gameWithState("Live"), reportKindScouting, "scouting", true},
		{"final routes to recap", true, gameWithState("Final"), reportKindRecap, "recap", true},
		{"unknown state has no report", true, gameWithState("Delayed"), 0, "", false},
		{"empty state has no report", true, gameWithState(""), 0, "", false},
		{"disabled config yields no report", false, gameWithState("Live"), 0, "", false},
		{"nil game yields no report", true, nil, 0, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kind, label, ok := reportKindFor(tc.enabled, tc.game)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !tc.wantOK {
				return
			}
			if kind != tc.wantKind {
				t.Errorf("kind = %v, want %v", kind, tc.wantKind)
			}
			if label != tc.wantLabel {
				t.Errorf("label = %q, want %q", label, tc.wantLabel)
			}
		})
	}
}

// gameData status must win over the schedule status when both are present.
func TestReportKindFor_PrefersGameDataState(t *testing.T) {
	g := &api.Game{
		Status:   api.GameStatus{AbstractGameState: "Preview"},
		GameData: &api.GameData{Status: api.GameStatus{AbstractGameState: "Live"}},
	}
	kind, _, ok := reportKindFor(true, g)
	if !ok || kind != reportKindScouting {
		t.Errorf("hydrated Live game should route to scouting, got kind=%v ok=%v", kind, ok)
	}
}
