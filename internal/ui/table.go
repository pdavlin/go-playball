package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderTable renders a table with headers and rows.
// widths: column widths. 0 means auto-calculate from content.
// maxWidth: if > 0, auto columns are shrunk so the total row fits.
// First column is left-aligned, all others right-aligned.
func renderTable(headers []string, widths []int, rows [][]string, maxWidth int) string {
	resolved := resolveWidths(headers, widths, rows, maxWidth)

	headerLine := formatRow(headers, resolved)
	hdrStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#000000"}).
		Background(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#FFFFFF"})

	lines := make([]string, 0, len(rows)+1)
	lines = append(lines, hdrStyle.Render(headerLine))
	for _, row := range rows {
		lines = append(lines, formatRow(row, resolved))
	}

	return strings.Join(lines, "\n")
}

// resolveWidths replaces 0-width entries with auto-calculated values.
// If maxWidth > 0, shrinks auto columns so total row width fits.
func resolveWidths(headers []string, widths []int, rows [][]string, maxWidth int) []int {
	resolved := make([]int, len(widths))
	for i, w := range widths {
		if w > 0 {
			resolved[i] = w
			continue
		}
		maxLen := len(headers[i])
		for _, row := range rows {
			if i < len(row) && len(row[i]) > maxLen {
				maxLen = len(row[i])
			}
		}
		resolved[i] = maxLen
	}

	if maxWidth > 0 {
		spaces := len(resolved) - 1
		total := spaces
		for _, w := range resolved {
			total += w
		}
		if total > maxWidth {
			excess := total - maxWidth
			for i := range widths {
				if widths[i] == 0 && excess > 0 {
					minW := len(headers[i])
					if minW < 6 {
						minW = 6
					}
					canShrink := resolved[i] - minW
					if canShrink < 0 {
						canShrink = 0
					}
					shrink := excess
					if shrink > canShrink {
						shrink = canShrink
					}
					resolved[i] -= shrink
					excess -= shrink
				}
			}
		}
	}

	return resolved
}

// formatRow formats a single row with column alignment.
// First column left-aligned, rest right-aligned.
// Cells wider than their column width are truncated.
func formatRow(cells []string, widths []int) string {
	parts := make([]string, len(cells))
	for i, cell := range cells {
		w := widths[i]
		if len(cell) > w {
			cell = cell[:w]
		}
		if i == 0 {
			parts[i] = fmt.Sprintf("%-*s", w, cell)
		} else {
			parts[i] = fmt.Sprintf("%*s", w, cell)
		}
	}
	return strings.Join(parts, " ")
}
