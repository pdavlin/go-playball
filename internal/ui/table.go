package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// renderTable renders a table with headers and rows.
// widths: column widths. 0 means auto-calculate from content.
// maxWidth: if > 0, auto columns are shrunk so the total row fits.
// First column is left-aligned, all others right-aligned.
func renderTable(headers []string, widths []int, rows [][]string, maxWidth int) string {
	resolved := resolveWidths(headers, widths, rows, maxWidth)
	return renderTableWithWidths(headers, resolved, rows)
}

// renderTablePair renders two related tables (e.g. an away/home pair shown
// side by side) sharing identical column geometry. Widths are resolved once
// from the combined content of both row sets so headers and stat columns
// land at the same x position in both tables, regardless of which side has
// the longer names.
func renderTablePair(headers []string, widths []int, rowsA, rowsB [][]string, maxWidth int) (string, string) {
	combined := make([][]string, 0, len(rowsA)+len(rowsB))
	combined = append(combined, rowsA...)
	combined = append(combined, rowsB...)
	resolved := resolveWidths(headers, widths, combined, maxWidth)
	return renderTableWithWidths(headers, resolved, rowsA), renderTableWithWidths(headers, resolved, rowsB)
}

// renderTableWithWidths renders headers and rows using already-resolved
// column widths.
func renderTableWithWidths(headers []string, resolved []int, rows [][]string) string {
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
		maxLen := ansi.StringWidth(headers[i])
		for _, row := range rows {
			if i < len(row) {
				if w := ansi.StringWidth(row[i]); w > maxLen {
					maxLen = w
				}
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
					minW := ansi.StringWidth(headers[i])
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
// Cells wider than their column width are truncated with a trailing
// ellipsis. Width measurement and truncation are ansi/rune aware, so
// multi-byte characters (e.g. accented names) and any embedded ANSI
// styling are handled without being cut mid-sequence.
func formatRow(cells []string, widths []int) string {
	parts := make([]string, len(cells))
	for i, cell := range cells {
		w := widths[i]
		if w < 0 {
			w = 0
		}
		cell = ansi.Truncate(cell, w, "…")
		pad := w - ansi.StringWidth(cell)
		if pad < 0 {
			pad = 0
		}
		padding := strings.Repeat(" ", pad)
		if i == 0 {
			parts[i] = cell + padding
		} else {
			parts[i] = padding + cell
		}
	}
	return strings.Join(parts, " ")
}
