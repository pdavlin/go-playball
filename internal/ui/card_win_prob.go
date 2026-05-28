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
	winProbSparkRows      = 6
)

type winProbSwing struct {
	delta float64 // home delta, signed
	team  string  // "home" | "away"
	event string  // play description / event name
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
				event := play.Result.Event
				if play.Result.Description != "" {
					event = play.Result.Description
				}
				swings = append(swings, winProbSwing{delta: delta, team: team, event: event})
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
	spark := renderWinProbSparkline(series, width, height)
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

func renderWinProbSparkline(series []float64, width, height int) string {
	sparkW := width
	if sparkW < 8 {
		sparkW = 8
	}
	rows := winProbSparkRows
	// Cap to ~half the available height so swings list has room.
	if rows > height/2 {
		rows = height / 2
	}
	if rows < 1 {
		rows = 1
	}

	style := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#1F6FEB", Dark: "#58A6FF"})
	return style.Render(renderSparklineMulti(series, sparkW, rows))
}

func renderWinProbSwings(swings []winProbSwing, awayAbbr, homeAbbr string,
	awayColors, homeColors TeamColors, width, height int) string {
	if len(swings) == 0 {
		return ""
	}

	// Show most recent N swings, newest first.
	n := len(swings)
	if n > winProbMaxSwings {
		swings = swings[n-winProbMaxSwings:]
	}

	headerStyleLocal := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}).
		Bold(true)

	var lines []string
	lines = append(lines, headerStyleLocal.Render("Recent swings"))

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
		prefix := fmt.Sprintf("%s +%.0f · ", teamStyle.Render(abbr), math.Abs(sw.delta))
		// Approximate prefix visible length: abbr (3) + " +N · " ~ 8-9.
		visiblePrefix := len(abbr) + 7
		eventLine := truncateOneLine(sw.event, width-visiblePrefix)
		lines = append(lines, prefix+eventLine)
	}

	return strings.Join(lines, "\n")
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
