package anim

import (
	"image/color"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var (
	red  = color.RGBA{255, 0, 0, 255}
	blue = color.RGBA{0, 0, 255, 255}
)

func TestMain(m *testing.M) {
	// Force TrueColor so lipgloss emits ANSI escape codes in CI/non-TTY.
	lipgloss.SetColorProfile(termenv.TrueColor)
	os.Exit(m.Run())
}

func TestGradientRamp_MultipleSteps(t *testing.T) {
	ramp := GradientRamp(red, blue, 5)
	if len(ramp) != 5 {
		t.Fatalf("expected 5 colors, got %d", len(ramp))
	}

	// First color should be red-ish (high R, low B)
	r0, _, b0, _ := ramp[0].RGBA()
	if r0>>8 < 200 || b0>>8 > 50 {
		t.Errorf("first color not red-ish: R=%d B=%d", r0>>8, b0>>8)
	}

	// Last color should be blue-ish (low R, high B)
	r4, _, b4, _ := ramp[4].RGBA()
	if r4>>8 > 50 || b4>>8 < 200 {
		t.Errorf("last color not blue-ish: R=%d B=%d", r4>>8, b4>>8)
	}
}

func TestGradientRamp_SingleStep(t *testing.T) {
	ramp := GradientRamp(red, blue, 1)
	if len(ramp) != 1 {
		t.Fatalf("expected 1 color, got %d", len(ramp))
	}

	// The single color should be the from color (red)
	r, _, b, _ := ramp[0].RGBA()
	if r>>8 < 200 || b>>8 > 50 {
		t.Errorf("single step should be from color (red): R=%d B=%d", r>>8, b>>8)
	}
}

func TestGradientRamp_ZeroSteps(t *testing.T) {
	ramp := GradientRamp(red, blue, 0)
	if ramp != nil {
		t.Fatalf("expected nil for 0 steps, got %v", ramp)
	}
}

func TestBlendGradient_EmptyString(t *testing.T) {
	result := BlendGradient("", red, blue)
	if result != "" {
		t.Fatalf("expected empty string, got %q", result)
	}
}

func TestBlendGradient_SingleChar(t *testing.T) {
	result := BlendGradient("A", red, blue)

	// Should contain ANSI escape sequences
	if !strings.Contains(result, "\x1b[") {
		t.Error("expected ANSI color codes in output")
	}

	// Visible width should be 1
	if w := lipgloss.Width(result); w != 1 {
		t.Errorf("expected visible width 1, got %d", w)
	}
}

func TestBlendGradient_MultiChar(t *testing.T) {
	result := BlendGradient("Hello", red, blue)

	// Should contain ANSI escape sequences
	if !strings.Contains(result, "\x1b[") {
		t.Error("expected ANSI color codes in output")
	}

	// Visible width should match input length
	if w := lipgloss.Width(result); w != 5 {
		t.Errorf("expected visible width 5, got %d", w)
	}
}

func TestBlendGradientBold(t *testing.T) {
	result := BlendGradientBold("Hi", red, blue)

	// Should contain bold ANSI sequence
	if !strings.Contains(result, "\x1b[") {
		t.Error("expected ANSI sequences in output")
	}

	// Visible width should match input length
	if w := lipgloss.Width(result); w != 2 {
		t.Errorf("expected visible width 2, got %d", w)
	}
}
