package ui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pdavlin/go-playball/internal/api"
)

// testKeyMsg builds a tea.KeyMsg representing a single-character key
// press. Single-rune strings map to KeyRunes; the dispatch in
// handlePregameKeys compares against msg.String(), which yields the
// rune for KeyRunes.
func testKeyMsg(s string) tea.KeyMsg {
	return tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune(s)})
}

func TestEnsurePregameDataInitializes(t *testing.T) {
	m := Model{}
	data := m.ensurePregameData(123)
	if data == nil {
		t.Fatal("ensurePregameData returned nil")
	}
	if m.pregameData == nil {
		t.Fatal("pregameData map was not initialized")
	}
	if got := m.pregameData[123]; got != data {
		t.Fatal("ensurePregameData did not insert the entry")
	}
}

func TestEnsurePregameDataReturnsExisting(t *testing.T) {
	existing := &pregameGameData{
		pitcherDetail: &pitcherDetailPayload{loaded: true},
	}
	m := Model{pregameData: map[int]*pregameGameData{42: existing}}
	got := m.ensurePregameData(42)
	if got != existing {
		t.Fatal("ensurePregameData created a new entry instead of reusing")
	}
	if !got.pitcherDetail.loaded {
		t.Fatal("ensurePregameData clobbered existing payload")
	}
}

func TestPregameTabDefaultsToOverview(t *testing.T) {
	m := Model{}
	if m.pregameTab != PregameTabOverview {
		t.Fatalf("default pregame tab = %v, want Overview", m.pregameTab)
	}
}

func TestHandlePregameKeysTabSwitching(t *testing.T) {
	// Constructing a Model with a non-nil currentGame to satisfy the
	// fetch-cmd path, even though we won't execute the returned cmd.
	game := &api.Game{
		ID: 1,
		GameData: &api.GameData{
			ProbablePitchers: api.ProbablePitchers{
				Away: api.ProbablePitcher{ID: 0},
				Home: api.ProbablePitcher{ID: 0},
			},
		},
	}
	m := Model{currentGame: game, apiClient: api.NewClient()}

	updated, _ := m.handlePregameKeys(testKeyMsg("1"))
	if got := updated.(Model).pregameTab; got != PregameTabPitchers {
		t.Errorf("key 1: got tab %v, want Pitchers", got)
	}
}

// (Removed: no disabled pregame tabs remain after Phase 4.)

func TestHandlePregameKeysHotBatsTabActive(t *testing.T) {
	game := &api.Game{
		ID:       1,
		GameData: &api.GameData{},
	}
	m := Model{currentGame: game, apiClient: api.NewClient()}
	updated, _ := m.handlePregameKeys(testKeyMsg("2"))
	if got := updated.(Model).pregameTab; got != PregameTabHotBats {
		t.Errorf("key 2 should switch to Hot Bats; got %v", got)
	}
}

func TestHandlePregameKeysWindowSwitchOnlyInHotBats(t *testing.T) {
	game := &api.Game{
		ID:       1,
		GameData: &api.GameData{},
	}
	// On Pitchers tab, window keys should not change the active window.
	m := Model{
		currentGame:   game,
		apiClient:     api.NewClient(),
		pregameTab:    PregameTabPitchers,
		hotBatsWindow: HotBatsL7,
	}
	for _, k := range []string{",", "."} {
		updated, _ := m.handlePregameKeys(testKeyMsg(k))
		if got := updated.(Model).hotBatsWindow; got != HotBatsL7 {
			t.Errorf("key %q on Pitchers tab changed window to %v", k, got)
		}
	}
	// On Hot Bats tab, . cycles forward (L7 → L15) and , cycles back (L15 → L7).
	m.pregameTab = PregameTabHotBats
	updated, _ := m.handlePregameKeys(testKeyMsg("."))
	if got := updated.(Model).hotBatsWindow; got != HotBatsL15 {
		t.Errorf(". should cycle to L15, got %v", got)
	}
	m.hotBatsWindow = HotBatsL30
	updated, _ = m.handlePregameKeys(testKeyMsg("."))
	if got := updated.(Model).hotBatsWindow; got != HotBatsL7 {
		t.Errorf(". from L30 should wrap to L7, got %v", got)
	}
	m.hotBatsWindow = HotBatsL7
	updated, _ = m.handlePregameKeys(testKeyMsg(","))
	if got := updated.(Model).hotBatsWindow; got != HotBatsL30 {
		t.Errorf(", from L7 should wrap to L30, got %v", got)
	}
}

func TestPregameTabCycle(t *testing.T) {
	game := &api.Game{ID: 1, GameData: &api.GameData{}}
	m := Model{
		currentGame: game,
		apiClient:   api.NewClient(),
		pregameTab:  PregameTabPitchers,
	}
	updated, _ := m.handlePregameKeys(testKeyMsg("l"))
	if got := updated.(Model).pregameTab; got != PregameTabHotBats {
		t.Errorf("l from Pitchers should land on Hot Bats, got %v", got)
	}
	// With all four tabs enabled, h from Pitchers wraps backward to
	// the last enabled tab (Bullpen).
	updated, _ = m.handlePregameKeys(testKeyMsg("h"))
	if got := updated.(Model).pregameTab; got != PregameTabBullpen {
		t.Errorf("h from Pitchers should wrap to Bullpen, got %v", got)
	}
}

func TestFetchPitcherDetailSkipsWhenInFlight(t *testing.T) {
	game := &api.Game{
		ID:       7,
		GameData: &api.GameData{},
	}
	m := Model{
		currentGame: game,
		apiClient:   api.NewClient(),
		pregameData: map[int]*pregameGameData{
			7: {pitcherDetail: &pitcherDetailPayload{}},
		},
	}
	if cmd := m.fetchPitcherDetailIfNeeded(); cmd != nil {
		t.Fatal("expected nil cmd when fetch already in-flight")
	}
}

func TestFetchPitcherDetailSkipsWhenLoaded(t *testing.T) {
	game := &api.Game{
		ID:       7,
		GameData: &api.GameData{},
	}
	m := Model{
		currentGame: game,
		apiClient:   api.NewClient(),
		pregameData: map[int]*pregameGameData{
			7: {pitcherDetail: &pitcherDetailPayload{loaded: true}},
		},
	}
	if cmd := m.fetchPitcherDetailIfNeeded(); cmd != nil {
		t.Fatal("expected nil cmd when fetch already complete")
	}
}

func TestPitcherDetailPayloadCarriesErrors(t *testing.T) {
	// Sanity check that pitcherDetailPayload preserves per-side error
	// state independently — the renderer uses this to fall back per
	// column.
	p := pitcherDetailPayload{
		awayErr: errors.New("net down"),
		homeArsenal: []api.ArsenalPitch{
			{Code: "FF", Description: "Four-seam FB", UsagePct: 50, AvgVelocity: 95},
		},
		loaded: true,
	}
	if p.awayErr == nil {
		t.Fatal("expected awayErr set")
	}
	if p.homeErr != nil {
		t.Fatal("expected homeErr nil")
	}
	if len(p.homeArsenal) != 1 {
		t.Fatalf("home arsenal len = %d, want 1", len(p.homeArsenal))
	}
}
