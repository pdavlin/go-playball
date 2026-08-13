package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/pdavlin/go-playball/internal/api"
)

// F4: the three final-game header cards must always fit the terminal.
func TestScoreboardCardWidthsFitTerminal(t *testing.T) {
	for width := 40; width <= 160; width++ {
		team, score, stacked := scoreboardCardWidths(width)
		if team < 1 || score < 1 {
			t.Fatalf("width %d: non-positive card widths (%d, %d)", width, team, score)
		}
		if stacked {
			// Cards sit one above another; each needs width+2 columns.
			if team+2 > width || score+2 > width {
				t.Errorf("width %d: stacked cards overflow (team %d, score %d)", width, team, score)
			}
			continue
		}
		row := 2*(team+2) + score + 2
		if row > width {
			t.Errorf("width %d: three-across row is %d columns", width, row)
		}
		if team < scoreboardMinTeamCardWidth || score < scoreboardMinScoreCardWidth {
			t.Errorf("width %d: cards below minimum (team %d, score %d)", width, team, score)
		}
	}
}

func TestScoreboardCardWidthsKeepFullSizeWhenWide(t *testing.T) {
	for _, width := range []int{86, 100, 120, 200} {
		team, score, stacked := scoreboardCardWidths(width)
		if stacked {
			t.Errorf("width %d: unexpectedly stacked", width)
		}
		if team != scoreboardTeamCardWidth || score != scoreboardScoreCardWidth {
			t.Errorf("width %d: got (%d, %d), want (%d, %d)",
				width, team, score, scoreboardTeamCardWidth, scoreboardScoreCardWidth)
		}
	}
}

func TestScoreboardCardsStackOnlyWhenTooNarrow(t *testing.T) {
	if _, _, stacked := scoreboardCardWidths(80); stacked {
		t.Error("80 cols should still fit three cards across")
	}
	if _, _, stacked := scoreboardCardWidths(52); stacked {
		t.Error("52 cols should still fit the minimum cards across")
	}
	if _, _, stacked := scoreboardCardWidths(40); !stacked {
		t.Error("40 cols should stack the cards")
	}
}

// F7: the live linescore degrades spacing, then innings, never R/H/E.
func TestLiveLinescoreLayoutFitsAndKeepsRHE(t *testing.T) {
	for _, width := range []int{80, 100, 120} {
		for innings := 9; innings <= 18; innings++ {
			layout := liveLinescoreLayout(innings, width)
			avail := width - liveLeftColumnWidth(width)
			if got := layout.width(innings); got > avail {
				t.Errorf("width %d, %d innings: block is %d columns, %d available",
					width, innings, got, avail)
			}
			if layout.cellWidth < 2 || layout.gap < 2 {
				t.Errorf("width %d, %d innings: degraded past the floor (%+v)", width, innings, layout)
			}
		}
	}
}

func TestLiveLinescoreDegradationOrder(t *testing.T) {
	// 120 cols: nothing is given up.
	if got := liveLinescoreLayout(9, 120); got != (linescoreLayout{cellWidth: 3, gap: 3}) {
		t.Errorf("120 cols: got %+v, want full spacing", got)
	}
	// 80 cols: only the gap before R tightens; every inning survives.
	got := liveLinescoreLayout(9, 80)
	if got != (linescoreLayout{cellWidth: 3, gap: 2}) {
		t.Errorf("80 cols: got %+v, want the tightened gap", got)
	}
	// 80 cols in extras: cell padding goes next, still no innings lost.
	got = liveLinescoreLayout(12, 80)
	if got.cellWidth != 2 || got.elided() {
		t.Errorf("80 cols/12 innings: got %+v, want tight cells and no elision", got)
	}
	// Deep extras: the earliest innings finally drop behind a marker.
	got = liveLinescoreLayout(16, 80)
	if !got.elided() {
		t.Errorf("80 cols/16 innings: got %+v, want early innings elided", got)
	}
}

func TestCompactSituationKeepsErrorsAt80(t *testing.T) {
	m := newLinescoreTestModel(80)
	out := m.renderCompactGameSituation(m.currentGame)

	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("compact situation = %d lines, want 3", len(lines))
	}
	if !strings.Contains(lines[0], "R") || !strings.Contains(lines[0], "H") || !strings.Contains(lines[0], "E") {
		t.Errorf("header lost an R/H/E column: %q", lines[0])
	}
	for i, line := range lines {
		if w := lipgloss.Width(line); w > 80 {
			t.Errorf("line %d is %d columns wide, want <= 80", i, w)
		}
	}
	// The away row ends with runs, hits and errors.
	if !strings.HasSuffix(strings.TrimRight(stripANSI(lines[1]), " "), "4  8  1") {
		t.Errorf("away row lost its R/H/E values: %q", stripANSI(lines[1]))
	}
}

func TestLinescoreInningsElidesEarlyInnings(t *testing.T) {
	ls := testLinescore(12)
	layout := linescoreLayout{cellWidth: 2, gap: 2, firstInning: 3}
	got := linescoreInnings(ls, 12, "away", layout)
	if !strings.HasPrefix(got, " …") {
		t.Errorf("elided innings missing the marker: %q", got)
	}
	// marker + 9 remaining innings, 2 columns each
	if want := 2 * 10; lipgloss.Width(got) != want {
		t.Errorf("innings block = %d columns, want %d", lipgloss.Width(got), want)
	}
}

// F10: the stacked box score gives the focused panel the height.
func TestStackedPanelHeightsAccordion(t *testing.T) {
	heights := stackedPanelHeights(27, [4]int{12, 12, 6, 6}, 1)
	if heights[1] <= heights[0] {
		t.Errorf("focused panel not expanded: %v", heights)
	}
	for i, h := range heights {
		if i == 1 {
			continue
		}
		if h != collapsedPanelHeight {
			t.Errorf("panel %d height = %d, want %d", i, h, collapsedPanelHeight)
		}
	}
	if total := heights[0] + heights[1] + heights[2] + heights[3] + 8; total > 27 {
		t.Errorf("stack occupies %d lines, want <= 27", total)
	}
}

func TestStackedPanelHeightsNeverOverflow(t *testing.T) {
	for available := 12; available <= 60; available++ {
		for focused := 0; focused < 4; focused++ {
			heights := stackedPanelHeights(available, [4]int{20, 20, 20, 20}, focused)
			total := 8
			for _, h := range heights {
				if h < 3 {
					t.Fatalf("available %d: panel height %d too small", available, h)
				}
				total += h
			}
			if total > available && available >= 20 {
				t.Errorf("available %d, focused %d: stack is %d lines", available, focused, total)
			}
		}
	}
}

func TestStackedPanelHeightsCapsToContent(t *testing.T) {
	// A 4-line table should not stretch to fill a tall terminal.
	heights := stackedPanelHeights(60, [4]int{4, 4, 4, 4}, 0)
	if heights[0] != 5 {
		t.Errorf("focused height = %d, want 5 (title + 4 rows)", heights[0])
	}
}

// F9: single-column standings cards stay narrow enough to scan.
func TestStandingsCardWidthCapped(t *testing.T) {
	if got := standingsCardWidth(100); got != maxStandingsCardWidth {
		t.Errorf("100 cols: card width = %d, want %d", got, maxStandingsCardWidth)
	}
	if got := standingsCardWidth(50); got != 50 {
		t.Errorf("50 cols: card width = %d, want 50", got)
	}
}

func newLinescoreTestModel(width int) Model {
	game := &api.Game{
		LiveData: &api.LiveData{Linescore: testLinescore(9)},
	}
	game.Teams.Away.Team.Name = "Seattle Mariners"
	game.Teams.Home.Team.Name = "New York Yankees"
	return Model{width: width, height: 40, currentGame: game}
}

func testLinescore(innings int) api.Linescore {
	ls := api.Linescore{
		CurrentInning: innings,
		InningState:   "Top",
	}
	for i := 0; i < innings; i++ {
		runs := 0
		ls.Innings = append(ls.Innings, api.Inning{
			Num:  i + 1,
			Away: api.InningScore{Runs: &runs},
			Home: api.InningScore{Runs: &runs},
		})
	}
	ls.Teams.Away = api.LinescoreTeam{Runs: 4, Hits: 8, Errors: 1}
	ls.Teams.Home = api.LinescoreTeam{Runs: 2, Hits: 5, Errors: 0}
	return ls
}
