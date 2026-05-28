package ui

import (
	"testing"

	"github.com/pdavlin/go-playball/internal/api"
)

func pitch(name string, speed float64) api.PlayEvent {
	return api.PlayEvent{
		IsPitch: true,
		Details: api.EventDetails{
			PitchType: &api.PitchType{Code: name[:2], Description: name},
		},
		PitchData: &api.PitchData{StartSpeed: speed},
	}
}

func play(pitcherID int, events ...api.PlayEvent) api.Play {
	return api.Play{
		Matchup:    api.Matchup{Pitcher: api.Player{ID: pitcherID}},
		PlayEvents: events,
	}
}

func TestAggregatePitcherMix_EmptyGame(t *testing.T) {
	total, rows := aggregatePitcherMix(&api.Game{}, 42)
	if total != 0 || len(rows) != 0 {
		t.Fatalf("expected empty, got total=%d rows=%d", total, len(rows))
	}
}

func TestAggregatePitcherMix_NilGame(t *testing.T) {
	total, rows := aggregatePitcherMix(nil, 42)
	if total != 0 || rows != nil {
		t.Fatalf("expected nil/zero, got total=%d rows=%d", total, len(rows))
	}
}

func TestAggregatePitcherMix_HappyPath(t *testing.T) {
	g := &api.Game{
		LiveData: &api.LiveData{
			Plays: api.Plays{
				AllPlays: []api.Play{
					play(7,
						pitch("Four-Seam Fastball", 95.0),
						pitch("Four-Seam Fastball", 96.0),
						pitch("Slider", 85.0),
						pitch("Changeup", 88.0),
						pitch("Four-Seam Fastball", 94.0),
					),
				},
			},
		},
	}

	total, rows := aggregatePitcherMix(g, 7)
	if total != 5 {
		t.Fatalf("total: want 5, got %d", total)
	}
	if len(rows) != 3 {
		t.Fatalf("rows: want 3, got %d", len(rows))
	}
	if rows[0].name != "Four-Seam Fastball" || rows[0].count != 3 {
		t.Fatalf("first row should be fastball x3, got %+v", rows[0])
	}
	if rows[0].totalSpeed != 95.0+96.0+94.0 {
		t.Fatalf("fastball totalSpeed wrong: %f", rows[0].totalSpeed)
	}
}

func TestAggregatePitcherMix_FiltersOtherPitcher(t *testing.T) {
	g := &api.Game{
		LiveData: &api.LiveData{
			Plays: api.Plays{
				AllPlays: []api.Play{
					play(7, pitch("Four-Seam Fastball", 95.0)),
					play(9, pitch("Slider", 85.0), pitch("Slider", 84.0)),
					play(7, pitch("Four-Seam Fastball", 96.0)),
				},
			},
		},
	}

	total, rows := aggregatePitcherMix(g, 9)
	if total != 2 {
		t.Fatalf("total: want 2, got %d", total)
	}
	if len(rows) != 1 || rows[0].name != "Slider" || rows[0].count != 2 {
		t.Fatalf("expected only slider x2, got %+v", rows)
	}
}

func TestAggregatePitcherMix_SkipsNilPitchType(t *testing.T) {
	g := &api.Game{
		LiveData: &api.LiveData{
			Plays: api.Plays{
				AllPlays: []api.Play{
					play(7,
						api.PlayEvent{IsPitch: true, Details: api.EventDetails{}},
						pitch("Slider", 85.0),
					),
				},
			},
		},
	}

	total, rows := aggregatePitcherMix(g, 7)
	if total != 1 {
		t.Fatalf("total: want 1, got %d", total)
	}
	if len(rows) != 1 || rows[0].name != "Slider" {
		t.Fatalf("expected only slider, got %+v", rows)
	}
}

func TestAggregatePitcherMix_SkipsNonPitchEvents(t *testing.T) {
	g := &api.Game{
		LiveData: &api.LiveData{
			Plays: api.Plays{
				AllPlays: []api.Play{
					play(7,
						api.PlayEvent{
							IsPitch: false,
							Type:    "action",
							Details: api.EventDetails{
								PitchType: &api.PitchType{Description: "Four-Seam Fastball"},
							},
						},
						pitch("Slider", 85.0),
					),
				},
			},
		},
	}

	total, rows := aggregatePitcherMix(g, 7)
	if total != 1 || len(rows) != 1 || rows[0].name != "Slider" {
		t.Fatalf("expected only slider x1, got total=%d rows=%+v", total, rows)
	}
}

func TestAggregatePitcherMix_SkipsEmptyDescription(t *testing.T) {
	g := &api.Game{
		LiveData: &api.LiveData{
			Plays: api.Plays{
				AllPlays: []api.Play{
					play(7,
						api.PlayEvent{
							IsPitch:   true,
							Details:   api.EventDetails{PitchType: &api.PitchType{Code: "FF"}},
							PitchData: &api.PitchData{StartSpeed: 95.0},
						},
						pitch("Slider", 85.0),
					),
				},
			},
		},
	}

	total, _ := aggregatePitcherMix(g, 7)
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
}
