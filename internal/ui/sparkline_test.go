package ui

import (
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderSparkline_Empty(t *testing.T) {
	if got := renderSparkline(nil, 10); got != "" {
		t.Fatalf("nil values: want empty, got %q", got)
	}
	if got := renderSparkline([]float64{}, 10); got != "" {
		t.Fatalf("empty values: want empty, got %q", got)
	}
	if got := renderSparkline([]float64{1, 2, 3}, 0); got != "" {
		t.Fatalf("zero width: want empty, got %q", got)
	}
}

func TestRenderSparkline_SingleValue(t *testing.T) {
	got := renderSparkline([]float64{50}, 10)
	if n := utf8.RuneCountInString(got); n != 10 {
		t.Fatalf("rune count: want 10, got %d (%q)", n, got)
	}
	// All runes should be the lowest bar.
	for _, r := range got {
		if r != []rune(sparkChars)[0] {
			t.Fatalf("expected only lowest bar, got %q in %q", r, got)
		}
	}
}

func TestRenderSparkline_AllEqual(t *testing.T) {
	got := renderSparkline([]float64{42, 42, 42, 42}, 4)
	if n := utf8.RuneCountInString(got); n != 4 {
		t.Fatalf("rune count: want 4, got %d", n)
	}
	for _, r := range got {
		if r != []rune(sparkChars)[0] {
			t.Fatalf("expected only lowest bar, got %q", r)
		}
	}
}

func TestRenderSparkline_FullRange(t *testing.T) {
	runes := []rune(renderSparkline([]float64{0, 50, 100}, 3))
	if len(runes) != 3 {
		t.Fatalf("rune count: want 3, got %d", len(runes))
	}
	all := []rune(sparkChars)
	if runes[0] != all[0] {
		t.Fatalf("first rune: want %q, got %q", all[0], runes[0])
	}
	if runes[len(runes)-1] != all[len(all)-1] {
		t.Fatalf("last rune: want %q, got %q", all[len(all)-1], runes[len(runes)-1])
	}
}

func TestRenderSparkline_Downsample(t *testing.T) {
	values := []float64{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	got := renderSparkline(values, 5)
	if n := utf8.RuneCountInString(got); n != 5 {
		t.Fatalf("rune count: want 5, got %d", n)
	}
}

func TestRenderSparkline_Upsample(t *testing.T) {
	values := []float64{0, 100}
	got := renderSparkline(values, 8)
	if n := utf8.RuneCountInString(got); n != 8 {
		t.Fatalf("rune count: want 8, got %d", n)
	}
	runes := []rune(got)
	all := []rune(sparkChars)
	if runes[0] != all[0] {
		t.Fatalf("first rune: want %q, got %q", all[0], runes[0])
	}
	if runes[len(runes)-1] != all[len(all)-1] {
		t.Fatalf("last rune: want %q, got %q", all[len(all)-1], runes[len(runes)-1])
	}
}

func TestRenderBipolarSparkline_Shape(t *testing.T) {
	// 5 rows above + 1 axis + 5 rows below = 11 lines.
	out := renderBipolarSparkline(
		[]float64{50, 50, 50}, 3, 5,
		lipgloss.Color("#FF0000"),
		lipgloss.Color("#0000FF"),
		lipgloss.Color("#888888"),
	)
	lines := splitLines(out)
	if len(lines) != 11 {
		t.Fatalf("lines: want 11, got %d", len(lines))
	}
}

func TestRenderBipolarSparkline_HomeBelow(t *testing.T) {
	// 100% home WP for all values: bars hit max depth on the below side.
	out := renderBipolarSparkline(
		[]float64{100, 100, 100}, 3, 4,
		lipgloss.Color("#FF0000"),
		lipgloss.Color("#0000FF"),
		lipgloss.Color("#888888"),
	)
	lines := splitLines(out)
	// 4 above + axis + 4 below = 9 lines
	if len(lines) != 9 {
		t.Fatalf("lines: want 9, got %d", len(lines))
	}
	// Above rows should be all space.
	for i := 0; i < 4; i++ {
		if containsBlock(lines[i]) {
			t.Fatalf("above row %d should be empty, got %q", i, lines[i])
		}
	}
	// Below rows should contain full blocks.
	for i := 5; i < 9; i++ {
		if !containsBlock(lines[i]) {
			t.Fatalf("below row %d should have blocks, got %q", i, lines[i])
		}
	}
}

func TestRenderBipolarSparkline_AwayAbove(t *testing.T) {
	out := renderBipolarSparkline(
		[]float64{0, 0, 0}, 3, 4,
		lipgloss.Color("#FF0000"),
		lipgloss.Color("#0000FF"),
		lipgloss.Color("#888888"),
	)
	lines := splitLines(out)
	for i := 0; i < 4; i++ {
		if !containsBlock(lines[i]) {
			t.Fatalf("above row %d should have blocks, got %q", i, lines[i])
		}
	}
	for i := 5; i < 9; i++ {
		if containsBlock(lines[i]) {
			t.Fatalf("below row %d should be empty, got %q", i, lines[i])
		}
	}
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	out := []string{""}
	for _, r := range s {
		if r == '\n' {
			out = append(out, "")
		} else {
			out[len(out)-1] += string(r)
		}
	}
	return out
}

func containsBlock(s string) bool {
	for _, r := range s {
		if r == '█' {
			return true
		}
	}
	return false
}

func TestBucketize_ExactSize(t *testing.T) {
	in := []float64{1, 2, 3}
	out := bucketize(in, 3)
	for i, v := range in {
		if out[i] != v {
			t.Fatalf("bucket %d: want %f, got %f", i, v, out[i])
		}
	}
}
