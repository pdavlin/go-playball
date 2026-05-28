package ui

import (
	"testing"

	"github.com/pdavlin/go-playball/internal/api"
)

func wpPlay(home, away float64, event, desc string) api.WinProbPlay {
	return api.WinProbPlay{
		Result:             api.PlayResult{Event: event, Description: desc},
		HomeWinProbability: home,
		AwayWinProbability: away,
	}
}

func TestExtractWinProbSeries_Empty(t *testing.T) {
	series, swings := extractWinProbSeries(nil)
	if series != nil || len(swings) != 0 {
		t.Fatalf("expected empty, got series=%v swings=%+v", series, swings)
	}
}

func TestExtractWinProbSeries_AllZero(t *testing.T) {
	plays := []api.WinProbPlay{wpPlay(0, 0, "K", "K looking")}
	series, swings := extractWinProbSeries(plays)
	if len(series) != 0 || len(swings) != 0 {
		t.Fatalf("expected empty, got series=%v swings=%+v", series, swings)
	}
}

func TestExtractWinProbSeries_HomeSeries(t *testing.T) {
	plays := []api.WinProbPlay{
		wpPlay(50, 50, "K", ""),
		wpPlay(55, 45, "S", ""),
		wpPlay(58, 42, "BB", ""),
	}
	series, _ := extractWinProbSeries(plays)
	want := []float64{50, 55, 58}
	if len(series) != len(want) {
		t.Fatalf("len: want %d, got %d", len(want), len(series))
	}
	for i, v := range want {
		if series[i] != v {
			t.Fatalf("series[%d]: want %f, got %f", i, v, series[i])
		}
	}
}

func TestExtractWinProbSeries_SwingHome(t *testing.T) {
	plays := []api.WinProbPlay{
		wpPlay(50, 50, "S1", ""),
		wpPlay(62, 38, "HR", "Home run"),
	}
	_, swings := extractWinProbSeries(plays)
	if len(swings) != 1 {
		t.Fatalf("swings: want 1, got %d", len(swings))
	}
	if swings[0].team != "home" || swings[0].delta != 12 || swings[0].event != "Home run" {
		t.Fatalf("unexpected swing: %+v", swings[0])
	}
}

func TestExtractWinProbSeries_SwingAway(t *testing.T) {
	plays := []api.WinProbPlay{
		wpPlay(60, 40, "S1", ""),
		wpPlay(40, 60, "Double", "2B RBI"),
	}
	_, swings := extractWinProbSeries(plays)
	if len(swings) != 1 {
		t.Fatalf("swings: want 1, got %d", len(swings))
	}
	if swings[0].team != "away" || swings[0].delta != -20 {
		t.Fatalf("unexpected swing: %+v", swings[0])
	}
}

func TestExtractWinProbSeries_MultipleSwings(t *testing.T) {
	plays := []api.WinProbPlay{
		wpPlay(50, 50, "", ""),
		wpPlay(62, 38, "HR", "Home run"),
		wpPlay(60, 40, "", ""),
		wpPlay(45, 55, "2B", "Double"),
	}
	_, swings := extractWinProbSeries(plays)
	if len(swings) != 2 {
		t.Fatalf("swings: want 2, got %d", len(swings))
	}
	if swings[0].team != "home" || swings[1].team != "away" {
		t.Fatalf("unexpected order: %+v", swings)
	}
}

func TestExtractWinProbSeries_NoSwingBelowThreshold(t *testing.T) {
	plays := []api.WinProbPlay{
		wpPlay(50, 50, "S1", ""),
		wpPlay(53, 47, "S2", ""),
	}
	_, swings := extractWinProbSeries(plays)
	if len(swings) != 0 {
		t.Fatalf("expected no swings, got %+v", swings)
	}
}
