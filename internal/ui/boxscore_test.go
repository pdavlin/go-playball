package ui

import (
	"strings"
	"testing"

	"github.com/pdavlin/go-playball/internal/api"
)

// mkPitcher builds a minimal boxscore player with a pitching decision note.
func mkPitcher(id int, name, note string) api.BoxscorePlayer {
	return api.BoxscorePlayer{
		Person: api.Player{ID: id, FullName: name},
		Stats: api.PlayerStats{
			Pitching: &api.PitchingStats{Note: note},
		},
	}
}

// TestBuildPitcherRowsNoteParens guards against doubled parens when the
// API's Note field already comes wrapped, e.g. "(L, 4-1)" rather than
// the bare "L, 4-1" the rendering code used to assume.
func TestBuildPitcherRowsNoteParens(t *testing.T) {
	tests := []struct {
		name       string
		playerName string
		note       string
		want       string
	}{
		{"note already parenthesized", "Baz", "(L, 4-1)", "Baz (L, 4-1)"},
		{"bare note", "Foo", "W, 5-2", "Foo (W, 5-2)"},
		{"empty note", "Bar", "", "Bar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			team := api.BoxscoreTeam{
				Pitchers: []int{1},
				Players: map[string]api.BoxscorePlayer{
					"ID1": mkPitcher(1, tt.playerName, tt.note),
				},
			}
			rows := buildPitcherRows(team, nil)
			// buildPitcherRows always appends a trailing "Totals" row.
			if len(rows) != 2 {
				t.Fatalf("expected 2 rows (pitcher + totals), got %d", len(rows))
			}
			got := rows[0][0]
			if got != tt.want {
				t.Errorf("name = %q, want %q", got, tt.want)
			}
			if strings.Contains(got, "((") {
				t.Errorf("name contains doubled paren: %q", got)
			}
		})
	}
}
