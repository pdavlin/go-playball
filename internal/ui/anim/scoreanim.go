package anim

import (
	"image/color"
	"math/rand/v2"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	scoreAnimFPS        = 20
	scoreAnimFrames     = 30 // 1.5 seconds at 20fps
	scoreRevealStart    = 20 // left-to-right reveal begins at frame 20
)

// ScoreAnimTickMsg triggers the next score animation frame.
type ScoreAnimTickMsg struct{ ID int }

// ScoreAnim is a short-lived animation that cycles characters across the
// linescore row for whichever team scored. After totalFrames it stops and
// the caller resumes normal rendering.
type ScoreAnim struct {
	id          int
	state       SpinnerState
	frameIdx    int
	awayFrames  []string
	homeFrames  []string
	awayChanged bool
	homeChanged bool
}

var nextScoreAnimID int

// NewScoreAnim creates a score animation. lineWidth is the visible character
// width of the animated portion (innings + gap + R H E). Only the teams whose
// score changed get the cycling treatment. awayText/homeText are the plain-text
// score lines revealed left-to-right during the wipe phase.
func NewScoreAnim(
	awayChanged, homeChanged bool,
	lineWidth int,
	awayFrom, awayTo color.Color,
	homeFrom, homeTo color.Color,
	awayText, homeText string,
) *ScoreAnim {
	nextScoreAnimID++

	var awayFrames []string
	var homeFrames []string

	if awayChanged {
		awayFrames = prerenderLineFrames(lineWidth, awayFrom, awayTo, awayText)
	}
	if homeChanged {
		homeFrames = prerenderLineFrames(lineWidth, homeFrom, homeTo, homeText)
	}

	return &ScoreAnim{
		id:          nextScoreAnimID,
		state:       SpinnerPaused,
		awayFrames:  awayFrames,
		homeFrames:  homeFrames,
		awayChanged: awayChanged,
		homeChanged: homeChanged,
	}
}

// prerenderLineFrames generates cycling character frames spanning width chars.
// During the last (scoreAnimFrames - scoreRevealStart) frames, the real score
// text is progressively revealed from left to right, creating a wipe effect.
func prerenderLineFrames(width int, from, to color.Color, realText string) []string {
	ramp := GradientRamp(from, to, width)
	colors := make([]lipgloss.Color, width)
	for i, c := range ramp {
		colors[i] = lipgloss.Color(ColorToHex(c))
	}

	realRunes := []rune(realText)
	for len(realRunes) < width {
		realRunes = append(realRunes, ' ')
	}
	realRunes = realRunes[:width]

	scoreColor := lipgloss.AdaptiveColor{Light: "#666666", Dark: "#AAAAAA"}

	frames := make([]string, scoreAnimFrames)
	for f := range frames {
		var revealCols int
		if f >= scoreRevealStart {
			progress := f - scoreRevealStart + 1
			total := scoreAnimFrames - scoreRevealStart
			revealCols = progress * width / total
		}

		var b strings.Builder
		for i := 0; i < width; i++ {
			if i < revealCols {
				b.WriteString(lipgloss.NewStyle().
					Foreground(scoreColor).
					Render(string(realRunes[i])))
			} else {
				ch := cyclingChars[rand.IntN(len(cyclingChars))]
				b.WriteString(lipgloss.NewStyle().
					Foreground(colors[i]).
					Bold(true).
					Render(string(ch)))
			}
		}
		frames[f] = b.String()
	}
	return frames
}

// Start sets the animation to running and returns the first tick command.
func (a *ScoreAnim) Start() (*ScoreAnim, tea.Cmd) {
	a.state = SpinnerRunning
	a.frameIdx = 0
	return a, a.tick()
}

// Update advances the animation frame.
func (a *ScoreAnim) Update(msg tea.Msg) (*ScoreAnim, tea.Cmd) {
	tickMsg, ok := msg.(ScoreAnimTickMsg)
	if !ok || tickMsg.ID != a.id {
		return a, nil
	}
	if a.state == SpinnerPaused {
		return a, nil
	}
	a.frameIdx++
	if a.frameIdx >= scoreAnimFrames {
		a.state = SpinnerPaused
		return a, nil
	}
	return a, a.tick()
}

// AwayView returns the cycling frame for the away team line, or empty string
// if the away team is not animating.
func (a *ScoreAnim) AwayView() string {
	if a.awayChanged && a.state == SpinnerRunning && a.frameIdx < scoreAnimFrames {
		return a.awayFrames[a.frameIdx]
	}
	return ""
}

// HomeView returns the cycling frame for the home team line, or empty string
// if the home team is not animating.
func (a *ScoreAnim) HomeView() string {
	if a.homeChanged && a.state == SpinnerRunning && a.frameIdx < scoreAnimFrames {
		return a.homeFrames[a.frameIdx]
	}
	return ""
}

// AwayChanged reports whether the away team's score changed.
func (a *ScoreAnim) AwayChanged() bool { return a.awayChanged }

// HomeChanged reports whether the home team's score changed.
func (a *ScoreAnim) HomeChanged() bool { return a.homeChanged }

// IsActive returns true if the animation is still running.
func (a *ScoreAnim) IsActive() bool {
	return a.state == SpinnerRunning
}

func (a *ScoreAnim) tick() tea.Cmd {
	id := a.id
	return tea.Tick(time.Second/scoreAnimFPS, func(t time.Time) tea.Msg {
		return ScoreAnimTickMsg{ID: id}
	})
}
