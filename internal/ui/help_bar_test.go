package ui

import (
	"strings"
	"testing"

	"github.com/pdavlin/go-playball/internal/api"
)

// TestBuildHelpBarFitsAll checks that when everything fits, all items
// render in their original order joined by " | ".
func TestBuildHelpBarFitsAll(t *testing.T) {
	items := []helpItem{
		{"a: alpha", helpPriorityCore},
		{"b: beta", helpPriorityNav},
		{"c: gamma", helpPriorityConvenience},
	}
	want := "a: alpha | b: beta | c: gamma"
	got := buildHelpBar(items, 100)
	if got != want {
		t.Fatalf("buildHelpBar() = %q, want %q", got, want)
	}
}

// TestBuildHelpBarDropsWholeItems checks that when the bar is too
// narrow for everything, lower-priority items are dropped first and
// higher-priority survivors keep their original relative order even
// though a lower-priority item sits between them in the source list.
func TestBuildHelpBarDropsWholeItems(t *testing.T) {
	items := []helpItem{
		{"q: quit", helpPriorityCore},
		{"z: convenience", helpPriorityConvenience},
		{"n: nav", helpPriorityNav},
	}
	// Budget fits "q: quit" (7) + " | " (3) + "n: nav" (6) = 16, but not
	// also "z: convenience" (14 more).
	got := buildHelpBar(items, 16)
	want := "q: quit | n: nav"
	if got != want {
		t.Fatalf("buildHelpBar() = %q, want %q", got, want)
	}
	if strings.Contains(got, "convenience") {
		t.Fatalf("expected the low-priority item to be dropped, got %q", got)
	}
}

// TestBuildHelpBarNeverEmitsPartialItem verifies there is never a
// dangling fragment of an item (e.g. a trailing separator or a cut
// token) at any width, by scanning a range of budgets and confirming
// every rendered item text appears in full or not at all.
func TestBuildHelpBarNeverEmitsPartialItem(t *testing.T) {
	items := []helpItem{
		{"c: schedule", helpPriorityNav},
		{"s: standings", helpPriorityNav},
		{"hjkl/arrows: navigate", helpPriorityCore},
		{"enter: view game", helpPriorityCore},
		{"p/n: prev/next day", helpPriorityConvenience},
		{"t: today", helpPriorityConvenience},
		{"q: quit", helpPriorityCore},
	}

	for width := 0; width <= 120; width++ {
		got := buildHelpBar(items, width)

		if strings.HasPrefix(got, " | ") || strings.HasSuffix(got, " | ") || strings.Contains(got, " |  | ") {
			t.Fatalf("width %d: dangling separator in %q", width, got)
		}

		for _, part := range strings.Split(got, " | ") {
			if part == "" {
				continue
			}
			found := false
			for _, it := range items {
				if it.text == part {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("width %d: emitted fragment %q that isn't a whole item (full string: %q)", width, part, got)
			}
		}

		if len(got) > width {
			t.Fatalf("width %d: rendered help %q (len %d) exceeds budget", width, got, len(got))
		}
	}
}

// TestBuildHelpBarEmptyOnZeroWidth confirms the degenerate case
// doesn't panic and produces nothing.
func TestBuildHelpBarEmptyOnZeroWidth(t *testing.T) {
	items := []helpItem{{"q: quit", helpPriorityCore}}
	if got := buildHelpBar(items, 0); got != "" {
		t.Fatalf("buildHelpBar(width=0) = %q, want empty", got)
	}
}

// TestGameSubviewHelpItemsFinalDropsLiveTabs ensures a final game never
// advertises the live tab strip, in either BoxScoreSubview (the default
// landing subview for a final game) or GameStatusSubview (reachable via
// "g" even after the game has ended).
func TestGameSubviewHelpItemsFinalDropsLiveTabs(t *testing.T) {
	final := &api.Game{Status: api.GameStatus{AbstractGameState: "Final"}}

	for _, sub := range []GameSubview{BoxScoreSubview, GameStatusSubview} {
		m := Model{currentGame: final, gameSubview: sub}
		for _, it := range m.gameSubviewHelpItems() {
			if strings.Contains(it.text, "1-3") || strings.Contains(it.text, "1-4/hl") {
				t.Fatalf("subview %v: final game advertised dead tab-strip item %q", sub, it.text)
			}
		}
	}
}

// TestGameSubviewHelpItemsPregameDropsDeadKeys ensures a preview game
// doesn't advertise b/a/p, since renderGame's Preview branch always
// renders the pregame tabs regardless of gameSubview.
func TestGameSubviewHelpItemsPregameDropsDeadKeys(t *testing.T) {
	preview := &api.Game{Status: api.GameStatus{AbstractGameState: "Preview"}}
	m := Model{currentGame: preview, gameSubview: GameStatusSubview}

	for _, it := range m.gameSubviewHelpItems() {
		if strings.HasPrefix(it.text, "b:") || strings.HasPrefix(it.text, "a:") || strings.HasPrefix(it.text, "p:") {
			t.Fatalf("preview game advertised dead key item %q", it.text)
		}
	}
}
