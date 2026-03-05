package anim

import (
	"image/color"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
)

// GradientRamp returns a slice of colors blended in HCL space between from and to.
func GradientRamp(from, to color.Color, steps int) []color.Color {
	if steps <= 0 {
		return nil
	}
	if steps == 1 {
		return []color.Color{from}
	}

	c1, _ := colorful.MakeColor(from)
	c2, _ := colorful.MakeColor(to)

	ramp := make([]color.Color, steps)
	for i := range ramp {
		t := float64(i) / float64(steps-1)
		ramp[i] = c1.BlendHcl(c2, t).Clamped()
	}
	return ramp
}

// BlendGradient applies a horizontal HCL color gradient to text,
// coloring each rune with an interpolated foreground color.
func BlendGradient(text string, from, to color.Color) string {
	runes := []rune(text)
	if len(runes) == 0 {
		return ""
	}

	ramp := GradientRamp(from, to, len(runes))
	out := make([]byte, 0, len(text)*20)
	for i, r := range runes {
		styled := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorToHex(ramp[i]))).Render(string(r))
		out = append(out, styled...)
	}
	return string(out)
}

// BlendGradientBold is BlendGradient with bold applied to each cluster.
func BlendGradientBold(text string, from, to color.Color) string {
	runes := []rune(text)
	if len(runes) == 0 {
		return ""
	}

	ramp := GradientRamp(from, to, len(runes))
	out := make([]byte, 0, len(text)*24)
	for i, r := range runes {
		styled := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ColorToHex(ramp[i]))).Render(string(r))
		out = append(out, styled...)
	}
	return string(out)
}

// RenderGradientBorder wraps pre-rendered content in a border where each
// border character is colored along a diagonal gradient from top-left (from)
// to bottom-right (to). The content should already be rendered at the target
// width (e.g. via lipgloss with Width + Padding but no Border).
func RenderGradientBorder(content string, width int, from, to color.Color, border lipgloss.Border) string {
	contentLines := strings.Split(content, "\n")
	totalRows := len(contentLines) + 2
	totalCols := width + 2
	maxRow := float64(totalRows - 1)
	maxCol := float64(totalCols - 1)

	c1, _ := colorful.MakeColor(from)
	c2, _ := colorful.MakeColor(to)

	colorAt := func(row, col int) lipgloss.Color {
		var t float64
		switch {
		case maxRow == 0 && maxCol == 0:
			t = 0
		case maxRow == 0:
			t = float64(col) / maxCol
		case maxCol == 0:
			t = float64(row) / maxRow
		default:
			t = (float64(row)/maxRow + float64(col)/maxCol) / 2
		}
		return lipgloss.Color(c1.BlendHcl(c2, t).Clamped().Hex())
	}

	var b strings.Builder

	// Top border
	b.WriteString(lipgloss.NewStyle().Foreground(colorAt(0, 0)).Render(border.TopLeft))
	for c := 1; c <= width; c++ {
		b.WriteString(lipgloss.NewStyle().Foreground(colorAt(0, c)).Render(border.Top))
	}
	b.WriteString(lipgloss.NewStyle().Foreground(colorAt(0, totalCols-1)).Render(border.TopRight))
	b.WriteString("\n")

	// Content rows with vertical borders
	for r, line := range contentLines {
		row := r + 1
		b.WriteString(lipgloss.NewStyle().Foreground(colorAt(row, 0)).Render(border.Left))
		b.WriteString(line)
		b.WriteString(lipgloss.NewStyle().Foreground(colorAt(row, totalCols-1)).Render(border.Right))
		b.WriteString("\n")
	}

	// Bottom border
	lastRow := totalRows - 1
	b.WriteString(lipgloss.NewStyle().Foreground(colorAt(lastRow, 0)).Render(border.BottomLeft))
	for c := 1; c <= width; c++ {
		b.WriteString(lipgloss.NewStyle().Foreground(colorAt(lastRow, c)).Render(border.Bottom))
	}
	b.WriteString(lipgloss.NewStyle().Foreground(colorAt(lastRow, totalCols-1)).Render(border.BottomRight))

	return b.String()
}

// ColorToHex converts a color.Color to a hex string like "#RRGGBB".
func ColorToHex(c color.Color) string {
	cf, ok := colorful.MakeColor(c)
	if !ok {
		return "#FFFFFF"
	}
	return cf.Clamped().Hex()
}
