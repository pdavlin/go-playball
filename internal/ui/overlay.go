package ui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// overlay composites fg on top of bg at the cell coordinates (x, y).
// Both strings may contain ANSI styling; the splice is performed by
// visual column using ansi.Cut so the unaffected portions of each bg
// line keep their styles intact. fg lines that extend past the bg
// canvas are clipped. bg lines shorter than x are padded with spaces.
func overlay(bg, fg string, x, y int) string {
	if fg == "" {
		return bg
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")

	// fgWidth is the modal's visual width — assume all fg rows are the
	// same visual width (the lipgloss panel pads them).
	fgWidth := 0
	for _, l := range fgLines {
		if w := ansi.StringWidth(l); w > fgWidth {
			fgWidth = w
		}
	}

	out := make([]string, len(bgLines), len(bgLines)+len(fgLines))
	copy(out, bgLines)

	// Extend the bg vertically if fg overflows the bottom edge.
	for len(out) < y+len(fgLines) {
		out = append(out, "")
	}

	for i, fgLine := range fgLines {
		row := y + i
		bgLine := out[row]
		bgWidth := ansi.StringWidth(bgLine)

		// Left side of the bg line up to column x. Pad if the bg line
		// is shorter than x so the modal still sits where intended.
		var left string
		if x <= bgWidth {
			left = ansi.Cut(bgLine, 0, x)
		} else {
			left = bgLine + strings.Repeat(" ", x-bgWidth)
		}

		// Right side starts after the modal ends. If the bg line
		// doesn't reach that far, the right segment is empty.
		var right string
		if x+fgWidth < bgWidth {
			right = ansi.Cut(bgLine, x+fgWidth, bgWidth)
		}

		out[row] = left + fgLine + right
	}

	return strings.Join(out, "\n")
}
