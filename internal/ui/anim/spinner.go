package anim

import (
	"image/color"
	"math/rand/v2"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// clampVisible truncates s to at most n visual cells, preserving ANSI
// styling on the kept prefix.
func clampVisible(s string, n int) string {
	if ansi.StringWidth(s) <= n {
		return s
	}
	return ansi.Truncate(s, n, "")
}

const (
	SpinnerFPS        = 20
	ellipsisAnimSpeed = 8  // frames per ellipsis step
	prerenderedFrames = 60 // 3 seconds of unique frames at 20fps
)

var cyclingChars = []rune("0123456789abcdefABCDEF~!@#$%^&*()+= ")
var maskGlyphs = []rune("0123456789abcdefABCDEF~!@#$%^&*()+=")
var ellipsisSteps = []string{".", "..", "...", ""}

// Word-mask tuning: ghost words are runs of minWordRun..maxWordRun glyph
// cells separated by stable single spaces; each glyph cell re-rolls only
// every maskChurnPeriod frames (staggered per cell) so the fill reads as
// text-shaped shimmer rather than full-line noise.
const (
	minWordRun      = 3
	maxWordRun      = 7
	maskChurnPeriod = 4
)

// SpinnerTickMsg triggers the next animation frame.
type SpinnerTickMsg struct{ ID int }

// SpinnerState represents whether the spinner is running or paused.
type SpinnerState int

const (
	SpinnerRunning SpinnerState = iota
	SpinnerPaused
)

var nextSpinnerID int

// Spinner is a cycling character animation with pre-cached frames and
// staggered birth offsets. All frames are generated at construction time
// so View() is a single slice lookup.
type Spinner struct {
	id           int
	state        SpinnerState
	size         int
	label        string
	frames       []string // pre-rendered cycling frame strings
	birthOffsets []int    // frame index at which each char appears
	dotFrames    []string // pre-rendered frames for the birth window
	frameIdx     int
}

// NewSpinner creates a spinner with a static gradient.
func NewSpinner(size int, label string, from, to color.Color) *Spinner {
	return newSpinner(size, label, from, to, false, false)
}

// NewCyclingSpinner creates a spinner with a shifting gradient that moves
// through the text each frame, creating the wave/shimmer effect from crush.
func NewCyclingSpinner(size int, label string, from, to color.Color) *Spinner {
	return newSpinner(size, label, from, to, true, false)
}

// NewWordMaskSpinner creates a cycling spinner whose frames are shaped
// like ghost text: stable word-length runs of slowly churning glyphs
// separated by fixed spaces. Intended for full-line streaming fills
// where the animation stands in for text that hasn't arrived yet.
func NewWordMaskSpinner(size int, label string, from, to color.Color) *Spinner {
	return newSpinner(size, label, from, to, true, true)
}

func newSpinner(size int, label string, from, to color.Color, cycleColors, wordMask bool) *Spinner {
	nextSpinnerID++

	// For cycling colors, generate a wider ramp and shift offset each frame.
	// The ramp goes from->to->from->to so it wraps smoothly.
	var numFrames int
	var ramp []color.Color
	if cycleColors {
		ramp = makeCyclingRamp(size*3, from, to)
		numFrames = size * 2 // one full cycle through the ramp
	} else {
		ramp = GradientRamp(from, to, size)
		numFrames = prerenderedFrames
	}

	// mask[i] is false for the fixed space cells between ghost words;
	// churnPeriod controls how many frames each glyph cell holds its
	// rune. The plain modes use an all-glyph mask churning every frame.
	churnPeriod := 1
	pool := cyclingChars
	mask := make([]bool, size)
	if wordMask {
		churnPeriod = maskChurnPeriod
		pool = maskGlyphs
		for i := 0; i < size; {
			run := minWordRun + rand.IntN(maxWordRun-minWordRun+1)
			for j := 0; j < run && i < size; j++ {
				mask[i] = true
				i++
			}
			i++ // single space between ghost words
		}
	} else {
		for i := range mask {
			mask[i] = true
		}
	}
	if numFrames%churnPeriod != 0 {
		numFrames += churnPeriod - numFrames%churnPeriod
	}
	if numFrames < SpinnerFPS {
		numFrames = SpinnerFPS
	}

	// Random birth offsets: each position appears at a different frame.
	// Churn offsets stagger which frame each glyph cell re-rolls on.
	birthOffsets := make([]int, size)
	churnOffsets := make([]int, size)
	for i := range birthOffsets {
		birthOffsets[i] = rand.IntN(SpinnerFPS)
		churnOffsets[i] = rand.IntN(churnPeriod)
	}

	// Pre-render cycling frames and the birth-window frames in one pass
	// so both share the same glyph churn state.
	frames := make([]string, numFrames)
	dotFrames := make([]string, SpinnerFPS)
	dotColor := lipgloss.AdaptiveColor{Light: "#666666", Dark: "#6272A4"}
	dot := lipgloss.NewStyle().Foreground(dotColor).Render(".")
	glyphs := make([]rune, size)
	offset := 0
	for f := range frames {
		var b, d strings.Builder
		for i := 0; i < size; i++ {
			if !mask[i] {
				b.WriteByte(' ')
				if f < SpinnerFPS {
					d.WriteByte(' ')
				}
				continue
			}
			if f == 0 || (f+churnOffsets[i])%churnPeriod == 0 {
				glyphs[i] = pool[rand.IntN(len(pool))]
			}
			idx := (i + offset) % len(ramp)
			styled := lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorToHex(ramp[idx]))).
				Render(string(glyphs[i]))
			b.WriteString(styled)
			if f < SpinnerFPS {
				if birthOffsets[i] > f {
					d.WriteString(dot)
				} else {
					d.WriteString(styled)
				}
			}
		}
		frames[f] = b.String()
		if f < SpinnerFPS {
			dotFrames[f] = d.String()
		}
		if cycleColors {
			offset++
		}
	}

	return &Spinner{
		id:           nextSpinnerID,
		state:        SpinnerPaused,
		size:         size,
		label:        label,
		frames:       frames,
		birthOffsets: birthOffsets,
		dotFrames:    dotFrames,
	}
}

// makeCyclingRamp generates a gradient ramp that goes from->to->from->to
// for smooth wrapping when the offset shifts each frame.
func makeCyclingRamp(size int, from, to color.Color) []color.Color {
	quarter := size / 4
	if quarter < 1 {
		quarter = 1
	}
	ramp := make([]color.Color, 0, size)
	seg1 := GradientRamp(from, to, quarter)
	seg2 := GradientRamp(to, from, quarter)
	for len(ramp) < size {
		ramp = append(ramp, seg1...)
		ramp = append(ramp, seg2...)
	}
	return ramp[:size]
}

// Update advances the spinner frame if the tick ID matches.
func (s *Spinner) Update(msg tea.Msg) (*Spinner, tea.Cmd) {
	tickMsg, ok := msg.(SpinnerTickMsg)
	if !ok || tickMsg.ID != s.id {
		return s, nil
	}
	if s.state == SpinnerPaused {
		return s, nil
	}
	s.frameIdx++
	return s, s.Tick()
}

// TrailerView returns up to n cycling color-cycled chars sourced from the
// spinner's current frame. Use it to render a "render-ahead" placeholder
// at the tail of streaming text: as new tokens arrive, the trailer slides
// forward, giving the illusion of cycling chars resolving into real text.
// Returns an empty string when n <= 0 or the spinner has no frames.
func (s *Spinner) TrailerView(n int) string {
	if n <= 0 || len(s.frames) == 0 {
		return ""
	}
	idx := s.frameIdx
	if s.frameIdx < SpinnerFPS {
		idx = s.frameIdx % len(s.dotFrames)
		full := s.dotFrames[idx]
		return clampVisible(full, n)
	}
	full := s.frames[s.frameIdx%len(s.frames)]
	return clampVisible(full, n)
}

// LineView returns the current frame's cells in columns [start, end).
// Unlike TrailerView, the slice is anchored to absolute frame columns:
// as a text prefix grows and start advances, each remaining animated
// cell keeps its column, so streamed text appears to resolve *through*
// the animation instead of pushing it forward. Returns an empty string
// when the range is empty or past the spinner's width.
func (s *Spinner) LineView(start, end int) string {
	if len(s.frames) == 0 {
		return ""
	}
	if start < 0 {
		start = 0
	}
	if end > s.size {
		end = s.size
	}
	if start >= end {
		return ""
	}
	var full string
	if s.frameIdx < SpinnerFPS {
		full = s.dotFrames[s.frameIdx%len(s.dotFrames)]
	} else {
		full = s.frames[s.frameIdx%len(s.frames)]
	}
	return ansi.Cut(full, start, end)
}

// View returns the current spinner frame with animated label.
func (s *Spinner) View() string {
	ellipsis := ellipsisSteps[(s.frameIdx/ellipsisAnimSpeed)%len(ellipsisSteps)]
	label := s.label + ellipsis

	var chars string
	if s.frameIdx < SpinnerFPS {
		chars = s.dotFrames[s.frameIdx]
	} else {
		chars = s.frames[s.frameIdx%len(s.frames)]
	}

	return label + " " + chars
}

// Start sets the spinner to running and returns the first tick command.
func (s *Spinner) Start() (*Spinner, tea.Cmd) {
	s.state = SpinnerRunning
	s.frameIdx = 0
	return s, s.Tick()
}

// Pause stops the tick chain. The next Update returns nil cmd.
func (s *Spinner) Pause() *Spinner {
	s.state = SpinnerPaused
	return s
}

// State returns the current spinner state.
func (s *Spinner) State() SpinnerState {
	return s.state
}

// Tick returns a tea.Cmd that sends a SpinnerTickMsg after one frame interval.
func (s *Spinner) Tick() tea.Cmd {
	id := s.id
	return tea.Tick(time.Second/SpinnerFPS, func(t time.Time) tea.Msg {
		return SpinnerTickMsg{ID: id}
	})
}
