package ui

import (
	"testing"
	"unicode/utf8"
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

func TestBucketize_ExactSize(t *testing.T) {
	in := []float64{1, 2, 3}
	out := bucketize(in, 3)
	for i, v := range in {
		if out[i] != v {
			t.Fatalf("bucket %d: want %f, got %f", i, v, out[i])
		}
	}
}
