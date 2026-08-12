package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pdavlin/go-playball/internal/api"
)

const pitchMixMinPitches = 5

type pitchAgg struct {
	name       string
	count      int
	totalSpeed float64
	maxSpeed   float64
}

type recentPitch struct {
	code  string
	speed float64
}

// aggregatePitcherMix walks the play-by-play and tallies the named
// pitcher's pitches by pitch-type description.
func aggregatePitcherMix(game *api.Game, pitcherID int) (total int, rows []pitchAgg) {
	if game == nil || game.LiveData == nil || pitcherID == 0 {
		return
	}
	bucket := map[string]*pitchAgg{}
	for _, play := range game.LiveData.Plays.AllPlays {
		if play.Matchup.Pitcher.ID != pitcherID {
			continue
		}
		for _, ev := range play.PlayEvents {
			if !ev.IsPitch || ev.Details.PitchType == nil {
				continue
			}
			name := ev.Details.PitchType.Description
			if name == "" {
				continue
			}
			agg, ok := bucket[name]
			if !ok {
				agg = &pitchAgg{name: name}
				bucket[name] = agg
			}
			agg.count++
			total++
			if ev.PitchData != nil {
				agg.totalSpeed += ev.PitchData.StartSpeed
				if ev.PitchData.StartSpeed > agg.maxSpeed {
					agg.maxSpeed = ev.PitchData.StartSpeed
				}
			}
		}
	}
	for _, v := range bucket {
		rows = append(rows, *v)
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].count > rows[j].count
	})
	return
}

// renderPitchMixCard renders the right-column Pitch Mix card showing
// the current pitcher's repertoire across the game. Returns natural
// height; the caller fits it via renderTabBodyViewport.
func renderPitchMixCard(game *api.Game, width int) string {
	return buildPitchMixBody(game, width)
}

func buildPitchMixBody(game *api.Game, width int) string {
	if game == nil || game.LiveData == nil {
		return itemStyle.Render("Loading...")
	}
	play := game.LiveData.Plays.CurrentPlay
	if play == nil {
		return itemStyle.Render("No active at-bat")
	}
	pitcherID := play.Matchup.Pitcher.ID
	if pitcherID == 0 {
		return itemStyle.Render("Awaiting next pitcher")
	}

	pitcher, hand := lookupPitcher(game, pitcherID)
	total, rows := aggregatePitcherMix(game, pitcherID)

	var b strings.Builder

	b.WriteString(renderPitchMixHeader(pitcher, hand, play.Matchup.Pitcher.FullName, total))
	b.WriteString("\n\n")

	if total < pitchMixMinPitches {
		b.WriteString(itemStyle.Render("Insufficient data"))
		return b.String()
	}

	b.WriteString(renderPitchMixTable(rows, total))

	recent := collectRecentPitches(game, pitcherID, 12)
	if len(recent) > 0 {
		b.WriteString("\n\n")
		b.WriteString(renderRecentPitches(recent, width))
	}

	if fastest := fastestPitch(rows); fastest != "" {
		b.WriteString("\n\n")
		b.WriteString(fastest)
	}

	return b.String()
}

func fastestPitch(rows []pitchAgg) string {
	var best *pitchAgg
	for i := range rows {
		if rows[i].maxSpeed == 0 {
			continue
		}
		if best == nil || rows[i].maxSpeed > best.maxSpeed {
			best = &rows[i]
		}
	}
	if best == nil {
		return ""
	}
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"})
	valueStyle := lipgloss.NewStyle().Bold(true)
	return labelStyle.Render("Fastest: ") +
		valueStyle.Render(fmt.Sprintf("%s %.1f mph", best.name, best.maxSpeed))
}

// collectRecentPitches walks AllPlays chronologically and returns the
// last `n` pitches for the given pitcher in chronological order.
func collectRecentPitches(game *api.Game, pitcherID int, n int) []recentPitch {
	if game == nil || game.LiveData == nil {
		return nil
	}
	var all []recentPitch
	for _, play := range game.LiveData.Plays.AllPlays {
		if play.Matchup.Pitcher.ID != pitcherID {
			continue
		}
		for _, ev := range play.PlayEvents {
			if !ev.IsPitch || ev.Details.PitchType == nil {
				continue
			}
			code := ev.Details.PitchType.Code
			if code == "" {
				continue
			}
			speed := 0.0
			if ev.PitchData != nil {
				speed = ev.PitchData.StartSpeed
			}
			all = append(all, recentPitch{code: code, speed: speed})
		}
	}
	if len(all) > n {
		all = all[len(all)-n:]
	}
	return all
}

func renderRecentPitches(pitches []recentPitch, width int) string {
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}).
		Bold(true)
	codeStyle := lipgloss.NewStyle().Bold(true)
	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#777777", Dark: "#999999"})
	sep := dimStyle.Render(" · ")

	var b strings.Builder
	b.WriteString(labelStyle.Render("Recent"))
	b.WriteString("\n")

	// Greedy wrap; track visible-length so styled text doesn't break it.
	lineLen := 0
	for i, p := range pitches {
		token := p.code
		if p.speed > 0 {
			token = fmt.Sprintf("%s %.0f", p.code, p.speed)
		}
		styled := codeStyle.Render(token)

		addLen := len(token)
		if i > 0 {
			addLen += 3 // " · "
		}
		if lineLen+addLen > width && lineLen > 0 {
			b.WriteString("\n")
			lineLen = 0
		}
		if lineLen > 0 {
			b.WriteString(sep)
			lineLen += 3
		}
		b.WriteString(styled)
		lineLen += len(token)
	}
	return b.String()
}

func renderPitchMixHeader(pitcher *api.BoxscorePlayer, hand, fallbackName string, total int) string {
	name := fallbackName
	if pitcher != nil && pitcher.Person.FullName != "" {
		name = pitcher.Person.FullName
	}

	nameStyle := lipgloss.NewStyle().Bold(true)
	subStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#AAAAAA"})

	parts := []string{nameStyle.Render(name)}
	if hand != "" {
		parts = append(parts, subStyle.Render("("+hand+"HP)"))
	}
	parts = append(parts, subStyle.Render(fmt.Sprintf("%d pitches", total)))

	return headerStyle.Render("Pitch Mix") + "\n" + strings.Join(parts, " ")
}

func renderPitchMixTable(rows []pitchAgg, total int) string {
	headerRow := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}).
		Bold(true).
		Render(fmt.Sprintf("%-18s %3s %6s %5s", "Pitch", "Cnt", "Pct", "Avg"))

	var b strings.Builder
	b.WriteString(headerRow)
	b.WriteString("\n")

	for _, r := range rows {
		pct := float64(r.count) * 100.0 / float64(total)
		avg := 0.0
		if r.count > 0 {
			avg = r.totalSpeed / float64(r.count)
		}
		avgStr := "  -"
		if avg > 0 {
			avgStr = fmt.Sprintf("%.1f", avg)
		}
		line := fmt.Sprintf("%-18s %3d %5.1f%% %5s", truncatePitchName(r.name, 18), r.count, pct, avgStr)
		b.WriteString(line)
		b.WriteString("\n")
	}

	return strings.TrimRight(b.String(), "\n")
}

func lookupPitcher(game *api.Game, pitcherID int) (*api.BoxscorePlayer, string) {
	key := fmt.Sprintf("ID%d", pitcherID)
	if p, ok := game.LiveData.Boxscore.Teams.Home.Players[key]; ok {
		return &p, ""
	}
	if p, ok := game.LiveData.Boxscore.Teams.Away.Players[key]; ok {
		return &p, ""
	}
	return nil, ""
}

func truncatePitchName(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}

// padToHeight truncates or pads the content to exactly `height` lines.
// Crucial: card renderers must not overflow their allocated height, or
// the surrounding layout scrolls the topRow off-screen.
func padToHeight(content string, height int) string {
	if height <= 0 {
		return ""
	}
	lines := strings.Split(content, "\n")
	if len(lines) > height {
		return strings.Join(lines[:height], "\n")
	}
	if len(lines) < height {
		return content + strings.Repeat("\n", height-len(lines))
	}
	return content
}
