package anim

import (
	"image/color"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ScrollDirection indicates whether the indicator points up or down.
type ScrollDirection int

const (
	ScrollUp ScrollDirection = iota
	ScrollDown
)

const (
	upTriangle       = "▲"
	downTriangle     = "▼"
	maxTriangles     = 3
	linesPerTriangle = 5
)

// ScrollIndicator returns a centered string of gradient-colored triangles
// indicating scrollable content in the given direction.
//
// remaining is the number of lines beyond the viewport edge.
// width is the available width for centering.
// from is the dim color (near viewport edge), to is the bright color (near content).
//
// Returns empty string if remaining <= 0.
func ScrollIndicator(dir ScrollDirection, remaining int, width int, from, to color.Color) string {
	if remaining <= 0 {
		return ""
	}

	count := remaining / linesPerTriangle
	if count < 1 {
		count = 1
	}
	if count > maxTriangles {
		count = maxTriangles
	}

	char := downTriangle
	if dir == ScrollUp {
		char = upTriangle
	}

	ramp := GradientRamp(from, to, count)
	parts := make([]string, count)
	for i := 0; i < count; i++ {
		parts[i] = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorToHex(ramp[i]))).
			Render(char)
	}

	indicator := strings.Join(parts, " ")
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(indicator)
}
