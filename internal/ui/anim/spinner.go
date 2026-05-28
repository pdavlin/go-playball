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
	ellipsisAnimSpeed = 8 // frames per ellipsis step
	prerenderedFrames = 60 // 3 seconds of unique frames at 20fps
)

var cyclingChars = []rune("0123456789abcdefABCDEF~!@#$%^&*()+= ")
var ellipsisSteps = []string{".", "..", "...", ""}

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
	return newSpinner(size, label, from, to, false)
}

// NewCyclingSpinner creates a spinner with a shifting gradient that moves
// through the text each frame, creating the wave/shimmer effect from crush.
func NewCyclingSpinner(size int, label string, from, to color.Color) *Spinner {
	return newSpinner(size, label, from, to, true)
}

func newSpinner(size int, label string, from, to color.Color, cycleColors bool) *Spinner {
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

	// Pre-render cycling frames
	frames := make([]string, numFrames)
	offset := 0
	for f := range frames {
		var b strings.Builder
		for i := 0; i < size; i++ {
			idx := i + offset
			if idx >= len(ramp) {
				idx = idx % len(ramp)
			}
			ch := cyclingChars[rand.IntN(len(cyclingChars))]
			b.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorToHex(ramp[idx]))).
				Render(string(ch)))
		}
		frames[f] = b.String()
		if cycleColors {
			offset++
		}
	}

	// Random birth offsets: each position appears at a different frame
	birthOffsets := make([]int, size)
	for i := range birthOffsets {
		birthOffsets[i] = rand.IntN(SpinnerFPS)
	}

	// Pre-render birth window frames (0..SpinnerFPS-1)
	dotFrames := make([]string, SpinnerFPS)
	dotColor := lipgloss.AdaptiveColor{Light: "#666666", Dark: "#6272A4"}
	for f := 0; f < SpinnerFPS; f++ {
		var b strings.Builder
		rampOffset := 0
		if cycleColors {
			rampOffset = f
		}
		for i := 0; i < size; i++ {
			if birthOffsets[i] > f {
				b.WriteString(lipgloss.NewStyle().Foreground(dotColor).Render("."))
			} else {
				idx := i + rampOffset
				if idx >= len(ramp) {
					idx = idx % len(ramp)
				}
				ch := cyclingChars[rand.IntN(len(cyclingChars))]
				b.WriteString(lipgloss.NewStyle().
					Foreground(lipgloss.Color(ColorToHex(ramp[idx]))).
					Render(string(ch)))
			}
		}
		dotFrames[f] = b.String()
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
