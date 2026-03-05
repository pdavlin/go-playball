package anim

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestScrollIndicator_ZeroRemaining(t *testing.T) {
	result := ScrollIndicator(ScrollDown, 0, 80, red, blue)
	if result != "" {
		t.Fatalf("expected empty string for 0 remaining, got %q", result)
	}
}

func TestScrollIndicator_NegativeRemaining(t *testing.T) {
	result := ScrollIndicator(ScrollDown, -5, 80, red, blue)
	if result != "" {
		t.Fatalf("expected empty string for negative remaining, got %q", result)
	}
}

func TestScrollIndicator_OneTriangle(t *testing.T) {
	result := ScrollIndicator(ScrollDown, 3, 80, red, blue)
	if !strings.Contains(result, downTriangle) {
		t.Error("expected down triangle in output")
	}
	count := strings.Count(result, downTriangle)
	if count != 1 {
		t.Errorf("expected 1 triangle, got %d", count)
	}
}

func TestScrollIndicator_TwoTriangles(t *testing.T) {
	result := ScrollIndicator(ScrollDown, 8, 80, red, blue)
	count := strings.Count(result, downTriangle)
	if count != 1 {
		t.Errorf("expected 1 triangle for 8 remaining (8/5=1), got %d", count)
	}
	result = ScrollIndicator(ScrollDown, 10, 80, red, blue)
	count = strings.Count(result, downTriangle)
	if count != 2 {
		t.Errorf("expected 2 triangles for 10 remaining (10/5=2), got %d", count)
	}
}

func TestScrollIndicator_CappedAtMax(t *testing.T) {
	result := ScrollIndicator(ScrollDown, 100, 80, red, blue)
	count := strings.Count(result, downTriangle)
	if count != maxTriangles {
		t.Errorf("expected %d triangles (capped), got %d", maxTriangles, count)
	}
}

func TestScrollIndicator_UpTriangle(t *testing.T) {
	result := ScrollIndicator(ScrollUp, 5, 80, red, blue)
	if !strings.Contains(result, upTriangle) {
		t.Error("expected up triangle in output")
	}
	if strings.Contains(result, downTriangle) {
		t.Error("should not contain down triangle for ScrollUp")
	}
}

func TestScrollIndicator_FitsWidth(t *testing.T) {
	result := ScrollIndicator(ScrollDown, 20, 40, red, blue)
	w := lipgloss.Width(result)
	if w > 40 {
		t.Errorf("expected visible width <= 40, got %d", w)
	}
}

func TestScrollIndicator_HasGradientColors(t *testing.T) {
	result := ScrollIndicator(ScrollDown, 15, 80, red, blue)
	if !strings.Contains(result, "\x1b[") {
		t.Error("expected ANSI color codes in output")
	}
}
