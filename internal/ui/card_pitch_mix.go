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
// the current pitcher's repertoire across the game.
func renderPitchMixCard(game *api.Game, height, width int) string {
	body := buildPitchMixBody(game, width)
	return padToHeight(body, height)
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

func padToHeight(content string, height int) string {
	if height <= 0 {
		return content
	}
	lines := strings.Count(content, "\n") + 1
	if lines >= height {
		return content
	}
	return content + strings.Repeat("\n", height-lines)
}
