package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pdavlin/go-playball/internal/api"
)

const (
	winProbSwingThreshold = 5.0
	winProbMaxSwings      = 6
	winProbSparkHalfRows  = 5
)

type winProbSwing struct {
	delta      float64 // home delta, signed
	team       string  // "home" | "away"
	halfInning string  // "top" | "bottom"
	inning     int
	batter     string
	event      string // result.event (e.g. "Single", "Home Run")
}

// extractWinProbSeries returns home-team WP percentages oldest-to-newest
// and the swings (>= threshold) in chronological order.
func extractWinProbSeries(plays []api.WinProbPlay) ([]float64, []winProbSwing) {
	var series []float64
	var swings []winProbSwing
	var prev float64
	for _, play := range plays {
		if play.HomeWinProbability == 0 && play.AwayWinProbability == 0 {
			continue
		}
		series = append(series, play.HomeWinProbability)

		if len(series) > 1 {
			delta := play.HomeWinProbability - prev
			if math.Abs(delta) >= winProbSwingThreshold {
				team := "home"
				if delta < 0 {
					team = "away"
				}
				swings = append(swings, winProbSwing{
					delta:      delta,
					team:       team,
					halfInning: play.About.HalfInning,
					inning:     play.About.Inning,
					batter:     play.Matchup.Batter.FullName,
					event:      play.Result.Event,
				})
			}
		}
		prev = play.HomeWinProbability
	}
	return series, swings
}

func renderWinProbCard(game *api.Game, plays []api.WinProbPlay, height, width int) string {
	body := buildWinProbBody(game, plays, height, width)
	return padToHeight(body, height)
}

func buildWinProbBody(game *api.Game, plays []api.WinProbPlay, height, width int) string {
	series, swings := extractWinProbSeries(plays)
	if len(series) < 3 {
		return itemStyle.Render("Win probability unavailable")
	}

	awayColors, homeColors := getGameTeamColorsFull(game)
	awayName := game.Teams.Away.Team.Name
	homeName := game.Teams.Home.Team.Name
	if awayName == "" && game.GameData != nil {
		awayName = game.GameData.Teams.Away.Name
	}
	if homeName == "" && game.GameData != nil {
		homeName = game.GameData.Teams.Home.Name
	}
	awayAbbr := getTeamAbbreviation(awayName)
	homeAbbr := getTeamAbbreviation(homeName)

	header := renderWinProbHeader(awayAbbr, homeAbbr, awayColors, homeColors, series)
	spark := renderWinProbChart(series, width, height, awayColors, homeColors)
	swingsList := renderWinProbSwings(swings, awayAbbr, homeAbbr, awayColors, homeColors, width, height)

	parts := []string{header, "", spark}
	if swingsList != "" {
		parts = append(parts, "", swingsList)
	}
	return strings.Join(parts, "\n")
}

func renderWinProbHeader(awayAbbr, homeAbbr string, away, home TeamColors, series []float64) string {
	homePct := series[len(series)-1]
	awayPct := 100 - homePct

	awayStyle := lipgloss.NewStyle().Foreground(away.Primary).Bold(true)
	homeStyle := lipgloss.NewStyle().Foreground(home.Primary).Bold(true)
	pctStyle := lipgloss.NewStyle().Bold(true)

	return fmt.Sprintf("%s %s    %s %s",
		awayStyle.Render(awayAbbr),
		pctStyle.Render(fmt.Sprintf("%2.0f%%", awayPct)),
		homeStyle.Render(homeAbbr),
		pctStyle.Render(fmt.Sprintf("%2.0f%%", homePct)),
	)
}

func renderWinProbChart(series []float64, width, height int,
	awayColors, homeColors TeamColors) string {
	sparkW := width
	if sparkW < 8 {
		sparkW = 8
	}

	halfRows := winProbSparkHalfRows
	// Total chart height is halfRows*2 + 1 axis. Cap to roughly half
	// the available height so the swings list still fits.
	maxHalf := (height - 1) / 4
	if halfRows > maxHalf {
		halfRows = maxHalf
	}
	if halfRows < 1 {
		halfRows = 1
	}

	axisColor := lipgloss.AdaptiveColor{Light: "#888888", Dark: "#666666"}
	return renderBipolarSparkline(series, sparkW, halfRows,
		awayColors.Primary, homeColors.Primary, axisColor)
}

func renderWinProbSwings(swings []winProbSwing, awayAbbr, homeAbbr string,
	awayColors, homeColors TeamColors, width, height int) string {
	if len(swings) == 0 {
		return ""
	}

	// Each swing renders as two lines, so cap to fit remaining height.
	maxSwings := winProbMaxSwings
	if maxSwings*2+1 > height {
		maxSwings = (height - 1) / 2
	}
	if maxSwings < 1 {
		return ""
	}
	if len(swings) > maxSwings {
		swings = swings[len(swings)-maxSwings:]
	}

	sectionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}).
		Bold(true)
	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#777777", Dark: "#999999"})
	eventStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#444444", Dark: "#CCCCCC"})

	var lines []string
	lines = append(lines, sectionStyle.Render("Big plays"))

	// Walk newest to oldest.
	for i := len(swings) - 1; i >= 0; i-- {
		sw := swings[i]
		abbr := homeAbbr
		colors := homeColors
		if sw.team == "away" {
			abbr = awayAbbr
			colors = awayColors
		}
		teamStyle := lipgloss.NewStyle().Foreground(colors.Primary).Bold(true)

		top := fmt.Sprintf("%s %s  %s",
			teamStyle.Render(abbr),
			lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("+%.0f", math.Abs(sw.delta))),
			dimStyle.Render(formatHalfInning(sw.halfInning, sw.inning)),
		)

		eventDesc := shortenSwingEvent(sw.batter, sw.event)
		bottom := eventStyle.Render(truncateOneLine(eventDesc, width))

		lines = append(lines, top, bottom)
	}

	return strings.Join(lines, "\n")
}

func formatHalfInning(half string, inning int) string {
	prefix := "TOP"
	if strings.EqualFold(half, "bottom") {
		prefix = "BOT"
	}
	return fmt.Sprintf("%s %d", prefix, inning)
}

// shortenSwingEvent builds the second-line description: batter name +
// lowercased event ("Shohei Ohtani home run").
func shortenSwingEvent(batter, event string) string {
	event = strings.ToLower(strings.TrimSpace(event))
	// Preserve common acronyms after lowercasing.
	for _, w := range []string{"dp", "hbp", "rbi"} {
		event = replaceWord(event, w, strings.ToUpper(w))
	}
	if batter == "" {
		return event
	}
	if event == "" {
		return batter
	}
	return batter + " " + event
}

func replaceWord(s, old, new string) string {
	parts := strings.Fields(s)
	for i, p := range parts {
		if p == old {
			parts[i] = new
		}
	}
	return strings.Join(parts, " ")
}

// truncateOneLine collapses to a single line and ellipsizes if needed.
func truncateOneLine(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if max < 4 {
		return ""
	}
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
