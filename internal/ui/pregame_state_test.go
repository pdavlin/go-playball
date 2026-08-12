package ui

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

func TestPregameTabDefaultsToPitchers(t *testing.T) {
	m := Model{}
	if m.pregameTab != PregameTabPitchers {
		t.Fatalf("default pregame tab = %v, want Pitchers", m.pregameTab)
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

func TestPregameViewportPadsShortContent(t *testing.T) {
	game := &api.Game{GameData: &api.GameData{}}
	out := renderTabBodyViewport(game, "a\nb", 40, 5, 0)
	if got := len(strings.Split(out, "\n")); got != 5 {
		t.Fatalf("padded viewport = %d lines, want 5", got)
	}
	if !strings.HasPrefix(out, "a\nb") {
		t.Fatalf("content not preserved: %q", out)
	}
}

func TestPregameViewportScrollsOverflow(t *testing.T) {
	game := &api.Game{GameData: &api.GameData{}}
	lines := make([]string, 12)
	for i := range lines {
		lines[i] = fmt.Sprintf("line%d", i)
	}
	body := strings.Join(lines, "\n")

	// At the top: no up indicator, first line visible, down indicator present.
	top := renderTabBodyViewport(game, body, 40, 5, 0)
	if !strings.Contains(top, "line0") {
		t.Errorf("top viewport missing first line: %q", top)
	}
	if strings.Contains(top, "line11") {
		t.Errorf("top viewport should not show last line: %q", top)
	}

	// Huge offset clamps to the bottom and shows the last line.
	bottom := renderTabBodyViewport(game, body, 40, 5, 9999)
	if !strings.Contains(bottom, "line11") {
		t.Errorf("bottom viewport missing last line: %q", bottom)
	}
	if got := len(strings.Split(bottom, "\n")); got != 5 {
		t.Errorf("bottom viewport = %d lines, want 5", got)
	}
}

func TestLiveTabCycle(t *testing.T) {
	game := &api.Game{ID: 1, GameData: &api.GameData{}}
	m := Model{currentGame: game, liveTab: LiveTabPlays, gameScrollOffset: 7}

	updated, _ := m.handleGameStatusKeys(testKeyMsg("l"))
	got := updated.(Model)
	if got.liveTab != LiveTabPitchMix {
		t.Errorf("l from Plays should land on Mix, got %v", got.liveTab)
	}
	if got.gameScrollOffset != 0 {
		t.Errorf("tab switch should reset scroll, got %d", got.gameScrollOffset)
	}

	m.liveTab = LiveTabPlays
	updated, _ = m.handleGameStatusKeys(testKeyMsg("h"))
	if got := updated.(Model).liveTab; got != LiveTabWinProb {
		t.Errorf("h from Plays should wrap to WinProb, got %v", got)
	}
}

// stripANSI removes color/style escape sequences so tests can assert
// on visible column positions.
func stripANSI(s string) string {
	return regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(s, "")
}

func TestUnderlineTabStripAlignment(t *testing.T) {
	entries := []stripEntry{
		{label: "Plays", selected: false},
		{label: "Pitch Mix", selected: true},
		{label: "Win Prob", selected: false},
	}
	out := renderUnderlineTabStrip(entries, 40, lipgloss.Color("#123456"))
	lines := strings.Split(out, "\n")
	if len(lines) != 2 {
		t.Fatalf("strip = %d lines, want 2", len(lines))
	}
	labelLine := stripANSI(lines[0])
	ruleLine := stripANSI(lines[1])

	// Compare rune columns, not byte offsets: the rule line's ─ glyphs
	// are multi-byte while the label line's padding is ASCII spaces.
	labelIdx := strings.Index(labelLine, "Pitch Mix")
	segIdx := strings.Index(ruleLine, "━")
	if labelIdx < 0 || segIdx < 0 {
		t.Fatalf("missing label or segment: %q / %q", labelLine, ruleLine)
	}
	labelStart := len([]rune(labelLine[:labelIdx]))
	segStart := len([]rune(ruleLine[:segIdx]))
	if labelStart != segStart {
		t.Errorf("underline segment at col %d, label at col %d", segStart, labelStart)
	}
	if got := strings.Count(ruleLine, "━"); got != len("Pitch Mix") {
		t.Errorf("segment width = %d, want %d", got, len("Pitch Mix"))
	}
	// The rule runs the full strip width.
	if got := len([]rune(ruleLine)); got != 40 {
		t.Errorf("rule width = %d runes, want 40", got)
	}
}
