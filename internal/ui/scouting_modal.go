package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/pdavlin/go-playball/internal/api"
	"github.com/pdavlin/go-playball/internal/scouting"
	"github.com/pdavlin/go-playball/internal/ui/anim"
)

// scoutingModal is the streaming-report overlay rendered above the schedule
// view when active.
type scoutingModal struct {
	gamePk       int
	matchupLabel string
	spinner      *anim.Spinner
	body         strings.Builder
	cached       bool
	cachedAt     time.Time
	err          error
	scrollOffset int
	streamDone   bool
	cancel       context.CancelFunc
	stream       <-chan scouting.Event
}

type scoutingEventMsg struct {
	gamePk int
	ev     scouting.Event
}

type scoutingClosedMsg struct {
	gamePk int
}

// openScoutingModal builds the modal, kicks off Generate, and returns the
// first command that drains one event from the stream.
func (m *Model) openScoutingModal(g *api.Game) tea.Cmd {
	if m.scoutingCache == nil {
		dir, err := scouting.DefaultCacheDir()
		if err == nil {
			m.scoutingCache = scouting.NewCache(dir)
		}
	}

	matchup := fmt.Sprintf("%s @ %s",
		shortOrName(g.Teams.Away.Team),
		shortOrName(g.Teams.Home.Team),
	)

	awayColors := GetTeamColors(g.Teams.Away.Team.Name)
	homeColors := GetTeamColors(g.Teams.Home.Team.Name)
	sp := anim.NewCyclingSpinner(15, "Scouting", awayColors.Primary, homeColors.Primary)
	sp, spinnerCmd := sp.Start()

	ctx, cancel := context.WithCancel(context.Background())
	stream, err := scouting.Generate(ctx, m.config.ScoutingValue(), m.scoutingCache, m.apiClient, g)
	if err != nil {
		cancel()
		modal := &scoutingModal{
			gamePk:       g.ID,
			matchupLabel: matchup,
			err:          err,
			streamDone:   true,
		}
		m.scoutingModal = modal
		return nil
	}

	modal := &scoutingModal{
		gamePk:       g.ID,
		matchupLabel: matchup,
		spinner:      sp,
		cancel:       cancel,
		stream:       stream,
	}
	m.scoutingModal = modal

	return tea.Batch(spinnerCmd, nextScoutingEvent(stream, g.ID))
}

func nextScoutingEvent(ch <-chan scouting.Event, gamePk int) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return scoutingClosedMsg{gamePk: gamePk}
		}
		return scoutingEventMsg{gamePk: gamePk, ev: ev}
	}
}

// applyEvent folds one event into the modal state.
func (mod *scoutingModal) applyEvent(ev scouting.Event) {
	switch ev.Kind {
	case scouting.EventDelta:
		mod.body.WriteString(ev.Text)
		if ev.Cached {
			mod.cached = true
			mod.cachedAt = ev.CachedAt
		}
	case scouting.EventDone:
		mod.streamDone = true
		if ev.Cached {
			mod.cached = true
			mod.cachedAt = ev.CachedAt
		}
		if mod.spinner != nil {
			mod.spinner = mod.spinner.Pause()
		}
	case scouting.EventError:
		mod.streamDone = true
		mod.err = ev.Err
		if mod.spinner != nil {
			mod.spinner = mod.spinner.Pause()
		}
	}
}

// handleKey processes a key while the modal owns the keyboard. The returned
// bool signals whether the parent should clear m.scoutingModal.
func (m *Model) handleScoutingKey(msg tea.KeyMsg) (close bool, cmd tea.Cmd) {
	mod := m.scoutingModal
	if mod == nil {
		return false, nil
	}
	switch msg.String() {
	case "esc", "q":
		if mod.cancel != nil {
			mod.cancel()
		}
		return true, nil
	case "R":
		if !mod.streamDone {
			return false, nil
		}
		// Find the game we opened on by gamePk so refresh works after
		// schedule re-fetches.
		game := m.findGameByID(mod.gamePk)
		if game == nil {
			return false, nil
		}
		_ = scouting.Delete(m.scoutingCache, mod.gamePk)
		if mod.cancel != nil {
			mod.cancel()
		}
		m.scoutingModal = nil
		return false, m.openScoutingModal(game)
	case "j", "down":
		mod.scrollOffset++
		return false, nil
	case "k", "up":
		if mod.scrollOffset > 0 {
			mod.scrollOffset--
		}
		return false, nil
	}
	return false, nil
}

func (m *Model) findGameByID(id int) *api.Game {
	for i := range m.games {
		if m.games[i].ID == id {
			return &m.games[i]
		}
	}
	return nil
}

// renderScoutingModal composes the modal panel and overlays it on top of
// bg (the underlying schedule view). Sizes the modal to min(80, width-4)
// x min(24, height-4) and centers it. ANSI styles in bg are preserved on
// the unaffected sides via overlay().
func (m Model) renderScoutingModal(bg string) string {
	mod := m.scoutingModal
	if mod == nil {
		return bg
	}

	outerW := minInt(80, m.width-4)
	if outerW < 30 {
		outerW = 30
	}
	outerH := minInt(24, m.height-4)
	if outerH < 8 {
		outerH = 8
	}

	innerW := outerW - 4 // padding(1,2) on both sides

	header := m.renderScoutingHeader(mod, innerW)
	body := m.renderScoutingBody(mod, innerW, outerH-3)

	content := header + "\n\n" + body
	panel := modalBorder.Width(outerW).Render(content)

	// Center: leave equal margin on left/top (rounded down).
	panelHeight := strings.Count(panel, "\n") + 1
	x := (m.width - outerW) / 2
	y := (m.height - panelHeight) / 2
	return overlay(bg, panel, x, y)
}

func (m Model) renderScoutingHeader(mod *scoutingModal, width int) string {
	title := fmt.Sprintf("Scouting · %s", mod.matchupLabel)
	switch {
	case mod.cached && mod.streamDone:
		suffix := modalDim.Render(fmt.Sprintf(" (cached %s)", relativeTime(mod.cachedAt)))
		return modalSectionHeader.Render(title) + suffix
	default:
		// The spinner has moved inline to the body trailer; the header
		// stays calm to keep focus on the streaming text.
		return modalSectionHeader.Render(title)
	}
}

// trailerLen is the visual width of the render-ahead spinner block that
// trails the latest streamed token.
const trailerLen = 10

func (m Model) renderScoutingBody(mod *scoutingModal, width, height int) string {
	if mod.err != nil {
		body := modalErrorText.Render(mod.err.Error())
		hint := modalDim.Render("press R to retry, esc to close")
		return body + "\n\n" + hint
	}
	raw := mod.body.String()
	streaming := !mod.streamDone && !mod.cached

	if raw == "" {
		if streaming && mod.spinner != nil {
			trailer := mod.spinner.TrailerView(trailerLen)
			return trailer
		}
		return modalDim.Render("waiting for first token…")
	}

	rendered := renderScoutingMarkdown(raw, width)
	if streaming && mod.spinner != nil {
		rendered = appendTrailer(rendered, mod.spinner.TrailerView(trailerLen), width)
	}

	lines := strings.Split(rendered, "\n")
	if mod.scrollOffset > len(lines)-height {
		mod.scrollOffset = len(lines) - height
	}
	if mod.scrollOffset < 0 {
		mod.scrollOffset = 0
	}
	end := mod.scrollOffset + height
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[mod.scrollOffset:end], "\n")
}

// appendTrailer appends trailer to the last line of rendered if it fits;
// otherwise wraps the trailer onto a new line. Width is the modal's inner
// content width.
func appendTrailer(rendered, trailer string, width int) string {
	if trailer == "" {
		return rendered
	}
	lines := strings.Split(rendered, "\n")
	last := lines[len(lines)-1]
	tw := visibleWidth(trailer)
	remaining := width - visibleWidth(last)
	switch {
	case remaining >= tw+1:
		lines[len(lines)-1] = last + " " + trailer
	case remaining >= tw:
		lines[len(lines)-1] = last + trailer
	default:
		// Trailer doesn't fit on this line — drop to next.
		lines = append(lines, trailer)
	}
	return strings.Join(lines, "\n")
}

// renderScoutingMarkdown is a deliberately minimal renderer: lines starting
// with "## " become bold-styled headers; everything else is wrapped to
// width. No external markdown library.
func renderScoutingMarkdown(s string, width int) string {
	if width < 10 {
		width = 10
	}
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(line, "## ") {
			out = append(out, modalSectionHeader.Render(strings.TrimPrefix(line, "## ")))
			continue
		}
		if line == "" {
			out = append(out, "")
			continue
		}
		for _, wrapped := range wrapLine(line, width) {
			out = append(out, wrapped)
		}
	}
	return strings.Join(out, "\n")
}

func wrapLine(line string, width int) []string {
	if len(line) <= width {
		return []string{line}
	}
	words := strings.Fields(line)
	if len(words) == 0 {
		return []string{line}
	}
	var rows []string
	var cur strings.Builder
	for _, w := range words {
		if cur.Len() == 0 {
			cur.WriteString(w)
			continue
		}
		if cur.Len()+1+len(w) > width {
			rows = append(rows, cur.String())
			cur.Reset()
			cur.WriteString(w)
			continue
		}
		cur.WriteByte(' ')
		cur.WriteString(w)
	}
	if cur.Len() > 0 {
		rows = append(rows, cur.String())
	}
	return rows
}

func relativeTime(t time.Time) string {
	if t.IsZero() {
		return "just now"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func visibleWidth(s string) int { return ansi.StringWidth(s) }

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func shortOrName(t api.Team) string {
	if t.Abbreviation != "" {
		return t.Abbreviation
	}
	return GetTeamShortName(t.Name)
}

// isPreviewGame reports whether g is in a state where scouting is meaningful.
func isPreviewGame(g *api.Game) bool {
	if g == nil {
		return false
	}
	return g.Status.AbstractGameState == "Preview"
}
