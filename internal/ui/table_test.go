package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestFormatRowTruncatesWithEllipsis(t *testing.T) {
	got := formatRow([]string{"Rodríguez, J (CF)"}, []int{10})
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("expected ellipsis-truncated cell, got %q", got)
	}
	if w := ansi.StringWidth(got); w != 10 {
		t.Fatalf("expected rendered width 10, got %d (%q)", w, got)
	}
	// Must not hard-cut a multi-byte rune: the string should stay valid UTF-8.
	if !strings.HasPrefix(got, "Rodr") {
		t.Fatalf("expected truncated name to keep leading runes intact, got %q", got)
	}
}

func TestFormatRowNoTruncationWhenItFits(t *testing.T) {
	got := formatRow([]string{"Ward, T (DH)"}, []int{20})
	if strings.Contains(got, "…") {
		t.Fatalf("did not expect truncation, got %q", got)
	}
	if w := ansi.StringWidth(got); w != 20 {
		t.Fatalf("expected padded width 20, got %d (%q)", w, got)
	}
}

func TestFormatRowRuneAwareWidth(t *testing.T) {
	// "í" is a single display column but 2 bytes in UTF-8; width accounting
	// must use display width, not byte length, both for padding and for
	// deciding whether truncation is needed at all.
	name := "Gómez, Y"
	if len(name) == ansi.StringWidth(name) {
		t.Fatalf("test fixture should differ in byte length vs display width")
	}
	got := formatRow([]string{name}, []int{12})
	if strings.Contains(got, "…") {
		t.Fatalf("name fits within width 12 by display width, should not truncate: %q", got)
	}
	if w := ansi.StringWidth(got); w != 12 {
		t.Fatalf("expected padded width 12, got %d (%q)", w, got)
	}
}

func TestRenderTablePairSharesColumnGeometry(t *testing.T) {
	headers := []string{"Pitchers", "IP", "ERA"}
	widths := []int{0, 4, 5}

	awayRows := [][]string{{"Miller, B", "5.0", "3.43"}}
	homeRows := [][]string{{"Blackburn, Preston", "0.2", "1.93"}}

	awayTable, homeTable := renderTablePair(headers, widths, awayRows, homeRows, 0)

	awayLines := strings.Split(awayTable, "\n")
	homeLines := strings.Split(homeTable, "\n")

	// Header rows must be identical length (shared geometry), and each
	// data row must align to the same total width as its header.
	if ansi.StringWidth(awayLines[0]) != ansi.StringWidth(homeLines[0]) {
		t.Fatalf("expected shared header width, got away=%d home=%d",
			ansi.StringWidth(awayLines[0]), ansi.StringWidth(homeLines[0]))
	}

	// The IP column (and everything after it) must start at the same index
	// in both tables' data rows since the name column width is shared.
	awayData := awayLines[1]
	homeData := homeLines[1]
	awayIdx := strings.Index(awayData, "5.0")
	homeIdx := strings.Index(homeData, "0.2")
	if awayIdx == -1 || homeIdx == -1 {
		t.Fatalf("expected to find IP values in rendered rows: away=%q home=%q", awayData, homeData)
	}
	if awayIdx != homeIdx {
		t.Fatalf("expected IP column to start at same x position, away=%d home=%d", awayIdx, homeIdx)
	}
}

func TestRenderTablePairMatchesUnpairedWhenGeometryAlreadyShared(t *testing.T) {
	headers := []string{"Batters", "AB"}
	widths := []int{0, 3}
	rows := [][]string{{"Ward, T", "4"}}

	paired, _ := renderTablePair(headers, widths, rows, rows, 0)
	single := renderTable(headers, widths, rows, 0)

	if paired != single {
		t.Fatalf("expected identical rendering when both sides share content:\npaired=%q\nsingle=%q", paired, single)
	}
}
