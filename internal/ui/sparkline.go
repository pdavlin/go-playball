package ui

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const sparkChars = "▁▂▃▄▅▆▇█"

// renderSparkline returns exactly `width` runes (each in [▁..█])
// representing the min-max-normalized series.
func renderSparkline(values []float64, width int) string {
	if width <= 0 || len(values) == 0 {
		return ""
	}
	runes := []rune(sparkChars)
	buckets := bucketize(values, width)

	min, max := buckets[0], buckets[0]
	for _, v := range buckets {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	span := max - min
	if span == 0 {
		return strings.Repeat(string(runes[0]), width)
	}

	var b strings.Builder
	for _, v := range buckets {
		idx := int(((v - min) / span) * float64(len(runes)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(runes) {
			idx = len(runes) - 1
		}
		b.WriteRune(runes[idx])
	}
	return b.String()
}

// renderSparklineMulti returns a `rows`-tall sparkline rendered with
// stacked block characters. Each column is `width` runes wide; the bar
// height in each column reflects min-max-normalized value.
func renderSparklineMulti(values []float64, width, rows int) string {
	if width <= 0 || rows <= 0 || len(values) == 0 {
		return ""
	}
	if rows == 1 {
		return renderSparkline(values, width)
	}

	buckets := bucketize(values, width)
	min, max := buckets[0], buckets[0]
	for _, v := range buckets {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	span := max - min

	partials := []rune(sparkChars)
	full := partials[len(partials)-1]
	totalLevels := rows * len(partials)

	lines := make([]strings.Builder, rows)
	for _, v := range buckets {
		var levels int
		if span == 0 {
			levels = 1
		} else {
			levels = int(((v - min) / span) * float64(totalLevels))
			if levels < 1 {
				levels = 1
			}
			if levels > totalLevels {
				levels = totalLevels
			}
		}
		for r := 0; r < rows; r++ {
			// Row 0 is the top of the bar; row rows-1 is the bottom.
			cellLevels := levels - (rows-1-r)*len(partials)
			switch {
			case cellLevels <= 0:
				lines[r].WriteRune(' ')
			case cellLevels >= len(partials):
				lines[r].WriteRune(full)
			default:
				lines[r].WriteRune(partials[cellLevels-1])
			}
		}
	}

	out := make([]string, rows)
	for i := range lines {
		out[i] = lines[i].String()
	}
	return strings.Join(out, "\n")
}

// renderBipolarSparkline draws a chart with a horizontal axis at 50.
// Values > 50 (home favored) push downward in `belowColor`; values <
// 50 (away favored) push upward in `aboveColor`. The bar magnitude is
// the absolute deviation from 50 on a fixed [0, 50] scale, so a 90-10
// game looks twice as decisive as a 70-30 game.
//
// Each side gets `halfRows` rows. Total chart height = halfRows*2 + 1.
// Bars are full-block-only for visual symmetry — Unicode lacks a
// top-anchored partial-fill set.
func renderBipolarSparkline(values []float64, width, halfRows int,
	aboveColor, belowColor, axisColor lipgloss.TerminalColor) string {
	if width <= 0 || halfRows <= 0 || len(values) == 0 {
		return ""
	}

	buckets := bucketize(values, width)
	rowPct := 50.0 / float64(halfRows)

	above := make([][]rune, halfRows)
	below := make([][]rune, halfRows)
	for i := range above {
		above[i] = make([]rune, width)
		below[i] = make([]rune, width)
		for j := range above[i] {
			above[i][j] = ' '
			below[i][j] = ' '
		}
	}

	for col, v := range buckets {
		delta := v - 50.0
		cells := int(math.Round(math.Abs(delta) / rowPct))
		if cells > halfRows {
			cells = halfRows
		}
		switch {
		case delta > 0:
			for r := 0; r < cells; r++ {
				below[r][col] = '█'
			}
		case delta < 0:
			for r := 0; r < cells; r++ {
				above[r][col] = '█'
			}
		}
	}

	aboveStyle := lipgloss.NewStyle().Foreground(aboveColor)
	belowStyle := lipgloss.NewStyle().Foreground(belowColor)
	axStyle := lipgloss.NewStyle().Foreground(axisColor)

	lines := make([]string, 0, halfRows*2+1)
	for r := halfRows - 1; r >= 0; r-- {
		lines = append(lines, aboveStyle.Render(string(above[r])))
	}
	lines = append(lines, axStyle.Render(strings.Repeat("─", width)))
	for r := 0; r < halfRows; r++ {
		lines = append(lines, belowStyle.Render(string(below[r])))
	}
	return strings.Join(lines, "\n")
}

// bucketize resamples `values` into exactly `width` buckets.
// Downsamples by averaging when len(values) > width; upsamples by
// linear interpolation when len(values) < width.
func bucketize(values []float64, width int) []float64 {
	if width <= 0 || len(values) == 0 {
		return nil
	}
	if len(values) == width {
		out := make([]float64, width)
		copy(out, values)
		return out
	}
	if len(values) == 1 {
		out := make([]float64, width)
		for i := range out {
			out[i] = values[0]
		}
		return out
	}

	out := make([]float64, width)

	if len(values) > width {
		// Downsample: average values in each bucket window.
		for i := 0; i < width; i++ {
			startF := float64(i) * float64(len(values)) / float64(width)
			endF := float64(i+1) * float64(len(values)) / float64(width)
			start := int(startF)
			end := int(endF)
			if end <= start {
				end = start + 1
			}
			if end > len(values) {
				end = len(values)
			}
			sum := 0.0
			for j := start; j < end; j++ {
				sum += values[j]
			}
			out[i] = sum / float64(end-start)
		}
		return out
	}

	// Upsample: linear interpolation.
	for i := 0; i < width; i++ {
		t := float64(i) * float64(len(values)-1) / float64(width-1)
		lo := int(t)
		hi := lo + 1
		if hi >= len(values) {
			out[i] = values[len(values)-1]
			continue
		}
		frac := t - float64(lo)
		out[i] = values[lo]*(1-frac) + values[hi]*frac
	}
	return out
}
