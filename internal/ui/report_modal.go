package ui

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/pdavlin/go-playball/internal/api"
	"github.com/pdavlin/go-playball/internal/recap"
	"github.com/pdavlin/go-playball/internal/scouting"
	"github.com/pdavlin/go-playball/internal/ui/anim"
)

// reportKind disambiguates the two streaming-report flavors the modal
// supports. They share rendering, scrolling, key handling, and event
// folding — only the source channel, the spinner label, and the title
// vary.
type reportKind int

const (
	reportKindScouting reportKind = iota
	reportKindRecap
)

// reportModal is the streaming-report overlay rendered above the
// schedule view. One struct handles both scouting and recap; the
// kind-specific stream is consumed via the nextEvent closure that the
// open* helpers install.
type reportModal struct {
	kind         reportKind
	gamePk       int
	matchupLabel string
	spinner      *anim.Spinner
	body         strings.Builder
	bodyRunes    int
	revealed     int
	cached       bool
	cachedAt     time.Time
	err          error
	scrollOffset int
	streamDone   bool
	cancel       context.CancelFunc
	nextEvent    func() tea.Cmd
}

// reportEventKind mirrors scouting.EventKind / recap.EventKind. The
// adapters at the source-package boundary translate to this so the
// modal's apply path is type-agnostic.
type reportEventKind int

const (
	reportEventDelta reportEventKind = iota
	reportEventDone
	reportEventError
)

type reportEvent struct {
	Kind     reportEventKind
	Text     string
	Err      error
	Cached   bool
	CachedAt time.Time
}

type reportEventMsg struct {
	gamePk int
	ev     reportEvent
}

type reportClosedMsg struct {
	gamePk int
}

func scoutingToReport(e scouting.Event) reportEvent {
	return reportEvent{
		Kind:     reportEventKind(e.Kind),
		Text:     e.Text,
		Err:      e.Err,
		Cached:   e.Cached,
		CachedAt: e.CachedAt,
	}
}

func recapToReport(e recap.Event) reportEvent {
	return reportEvent{
		Kind:     reportEventKind(e.Kind),
		Text:     e.Text,
		Err:      e.Err,
		Cached:   e.Cached,
		CachedAt: e.CachedAt,
	}
}

func nextScoutingEventCmd(ch <-chan scouting.Event, gamePk int) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return reportClosedMsg{gamePk: gamePk}
		}
		return reportEventMsg{gamePk: gamePk, ev: scoutingToReport(ev)}
	}
}

func nextRecapEventCmd(ch <-chan recap.Event, gamePk int) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return reportClosedMsg{gamePk: gamePk}
		}
		return reportEventMsg{gamePk: gamePk, ev: recapToReport(ev)}
	}
}

func (m *Model) openScoutingModal(g *api.Game) tea.Cmd {
	if m.scoutingCache == nil {
		dir, err := scouting.DefaultCacheDir()
		if err == nil {
			m.scoutingCache = scouting.NewCache(dir)
		}
	}
	return m.openReportModal(reportKindScouting, g, "Scouting", func(ctx context.Context) (tea.Cmd, error) {
		stream, err := scouting.Generate(ctx, m.config.ScoutingValue(), m.scoutingCache, m.apiClient, g)
		if err != nil {
			return nil, err
		}
		return nextScoutingEventCmd(stream, g.ID), nil
	})
}

func (m *Model) openRecapModal(g *api.Game) tea.Cmd {
	if m.recapCache == nil {
		dir, err := recap.DefaultCacheDir()
		if err == nil {
			m.recapCache = recap.NewCache(dir)
		}
	}
	return m.openReportModal(reportKindRecap, g, "Recapping", func(ctx context.Context) (tea.Cmd, error) {
		stream, err := recap.Generate(ctx, m.config.ScoutingValue(), m.recapCache, m.apiClient, g)
		if err != nil {
			return nil, err
		}
		return nextRecapEventCmd(stream, g.ID), nil
	})
}

// openReportModal builds the modal scaffold and starts the stream. The
// startStream closure decouples the modal layer from the choice of
// generator package.
func (m *Model) openReportModal(
	kind reportKind,
	g *api.Game,
	spinnerLabel string,
	startStream func(ctx context.Context) (tea.Cmd, error),
) tea.Cmd {
	matchup := fmt.Sprintf("%s @ %s",
		shortOrName(g.Teams.Away.Team),
		shortOrName(g.Teams.Home.Team),
	)

	awayColors := GetTeamColors(g.Teams.Away.Team.Name)
	homeColors := GetTeamColors(g.Teams.Home.Team.Name)
	sp := anim.NewWordMaskSpinner(reportAnimWidth, spinnerLabel, awayColors.Primary, homeColors.Primary)
	sp, spinnerCmd := sp.Start()

	ctx, cancel := context.WithCancel(context.Background())
	nextCmd, err := startStream(ctx)
	if err != nil {
		cancel()
		m.reportModal = &reportModal{
			kind:         kind,
			gamePk:       g.ID,
			matchupLabel: matchup,
			err:          err,
			streamDone:   true,
		}
		return nil
	}

	mod := &reportModal{
		kind:         kind,
		gamePk:       g.ID,
		matchupLabel: matchup,
		spinner:      sp,
		cancel:       cancel,
		nextEvent:    func() tea.Cmd { return nextCmd },
	}
	m.reportModal = mod

	return tea.Batch(spinnerCmd, nextCmd)
}

// applyEvent folds one event into the modal state.
func (mod *reportModal) applyEvent(ev reportEvent) {
	switch ev.Kind {
	case reportEventDelta:
		mod.body.WriteString(ev.Text)
		mod.bodyRunes += utf8.RuneCountInString(ev.Text)
		if ev.Cached {
			mod.cached = true
			mod.cachedAt = ev.CachedAt
			mod.revealed = mod.bodyRunes
		}
	case reportEventDone:
		// The spinner is left running: advanceReveal pauses it once the
		// reveal cursor catches up with the streamed text.
		mod.streamDone = true
		if ev.Cached {
			mod.cached = true
			mod.cachedAt = ev.CachedAt
			mod.revealed = mod.bodyRunes
		}
	case reportEventError:
		mod.streamDone = true
		mod.err = ev.Err
		mod.revealed = mod.bodyRunes
		if mod.spinner != nil {
			mod.spinner = mod.spinner.Pause()
		}
	}
}

// Reveal pacing: each spinner tick reveals revealMinStep runes plus a
// fraction of the backlog, so display trails arrival by a beat but
// catches up quickly after bursts.
const (
	revealMinStep    = 2
	revealCatchupDiv = 8
)

// advanceReveal moves the per-character reveal cursor toward the number
// of streamed runes. Called once per consumed spinner tick; pauses the
// spinner once the stream is done and the cursor has caught up.
func (mod *reportModal) advanceReveal() {
	backlog := mod.bodyRunes - mod.revealed
	if backlog <= 0 {
		if mod.streamDone && mod.spinner != nil {
			mod.spinner = mod.spinner.Pause()
		}
		return
	}
	step := revealMinStep + backlog/revealCatchupDiv
	if step > backlog {
		step = backlog
	}
	mod.revealed += step
}

// handleReportKey processes a key while the modal owns the keyboard.
// The returned bool signals whether the parent should clear m.reportModal.
func (m *Model) handleReportKey(msg tea.KeyMsg) (closeModal bool, cmd tea.Cmd) {
	mod := m.reportModal
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
		game := m.findGameByID(mod.gamePk)
		if game == nil {
			return false, nil
		}
		switch mod.kind {
		case reportKindScouting:
			_ = scouting.Delete(m.scoutingCache, mod.gamePk)
			if mod.cancel != nil {
				mod.cancel()
			}
			m.reportModal = nil
			return false, m.openScoutingModal(game)
		case reportKindRecap:
			_ = recap.Delete(m.recapCache, mod.gamePk)
			if mod.cancel != nil {
				mod.cancel()
			}
			m.reportModal = nil
			return false, m.openRecapModal(game)
		}
		return false, nil
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

// renderReportModal composes the modal panel and overlays it on top of
// bg (the underlying schedule view).
func (m Model) renderReportModal(bg string) string {
	mod := m.reportModal
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

	innerW := outerW - 4

	header := m.renderReportHeader(mod, innerW)
	body := m.renderReportBody(mod, innerW, outerH-3)

	content := header + "\n\n" + body
	panel := modalBorder.Width(outerW).Render(content)

	panelHeight := strings.Count(panel, "\n") + 1
	x := (m.width - outerW) / 2
	y := (m.height - panelHeight) / 2
	return overlay(bg, panel, x, y)
}

func (m Model) renderReportHeader(mod *reportModal, width int) string {
	prefix := "Scouting"
	if mod.kind == reportKindRecap {
		prefix = "Postgame"
	}
	title := fmt.Sprintf("%s · %s", prefix, mod.matchupLabel)
	if mod.cached && mod.streamDone {
		suffix := modalDim.Render(fmt.Sprintf(" (cached %s)", relativeTime(mod.cachedAt)))
		return modalSectionHeader.Render(title) + suffix
	}
	return modalSectionHeader.Render(title)
}

// reportAnimWidth is the pre-rendered width of the streaming animation.
// It matches the widest possible modal interior (outer 80 minus border
// and padding); narrower terminals slice a shorter range per render.
const reportAnimWidth = 76

func (m Model) renderReportBody(mod *reportModal, width, height int) string {
	if mod.err != nil {
		body := modalErrorText.Render(mod.err.Error())
		hint := modalDim.Render("press R to retry, esc to close")
		return body + "\n\n" + hint
	}
	raw := mod.body.String()
	animating := mod.spinner != nil && !mod.cached &&
		(!mod.streamDone || mod.revealed < mod.bodyRunes)
	if animating {
		raw = truncateRunes(raw, mod.revealed)
	}

	if raw == "" {
		if animating {
			return mod.spinner.LineView(0, width)
		}
		return modalDim.Render("waiting for first token…")
	}

	rendered := renderReportMarkdown(raw, width)
	if animating {
		rendered = fillLineWithAnim(rendered, mod.spinner, width)
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

// truncateRunes returns the prefix of s holding at most n runes.
func truncateRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	count := 0
	for i := range s {
		if count == n {
			return s[:i]
		}
		count++
	}
	return s
}

// fillLineWithAnim pads the last line of rendered with the spinner's
// current frame out to width. The fill is sliced at the column where
// the streamed text ends, so each animated cell holds its column while
// arriving tokens consume the line from the left.
func fillLineWithAnim(rendered string, sp *anim.Spinner, width int) string {
	lines := strings.Split(rendered, "\n")
	last := lines[len(lines)-1]
	fill := sp.LineView(visibleWidth(last), width)
	if fill == "" {
		return rendered
	}
	lines[len(lines)-1] = last + fill
	return strings.Join(lines, "\n")
}

// renderReportMarkdown is a deliberately minimal renderer: lines starting
// with "## " become bold-styled headers; everything else is wrapped to
// width. No external markdown library.
func renderReportMarkdown(s string, width int) string {
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

// reportKindFor returns the report kind appropriate for g, plus the
// help-bar label fragment ("scouting" / "recap"), or false if no report
// is available (game state mismatch or scouting not configured).
func reportKindFor(cfgEnabled bool, g *api.Game) (reportKind, string, bool) {
	if !cfgEnabled || g == nil {
		return 0, "", false
	}
	state := gameAbstractState(g)
	switch state {
	case "Preview":
		return reportKindScouting, "scouting", true
	case "Live":
		// In-progress games route to the scouting path, which selects its
		// in-progress tense from the live game state internally.
		return reportKindScouting, "scouting", true
	case "Final":
		return reportKindRecap, "recap", true
	default:
		return 0, "", false
	}
}

func gameAbstractState(g *api.Game) string {
	if g == nil {
		return ""
	}
	if g.GameData != nil && g.GameData.Status.AbstractGameState != "" {
		return g.GameData.Status.AbstractGameState
	}
	return g.Status.AbstractGameState
}
