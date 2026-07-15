package scouting

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pdavlin/go-playball/internal/api"
)

const systemPrompt = `You are a major-league baseball scout writing a tight,
pre-game scouting report for a fan who has thirty seconds. Output Markdown
with these literal section headers, in order:

## The Edge
## Pitching Edge
## Bats to Watch

After those three, you are encouraged to append additional ## sections
of your choosing at the bottom whenever the supplied context contains
something material a fan should know: weather or wind that changes run
scoring, a team riding a hot or cold streak, a batter whose recent form
diverges sharply from his season line, a starter trending up or down
across his last few outings, park quirks, or anything else concrete.
Examples: ## Weather, ## Hot Bats, ## Trending, ## X-Factor. Add as many
as the data supports, but every extra section must be grounded in a stat
or condition from the input — do not pad with generic filler.

Constraints:
- "## The Edge" must be exactly one sentence naming who has the advantage
  and why.
- "## Pitching Edge" compares the two starters in 2-4 short lines.
- "## Bats to Watch" lists 2-4 bullets, one batter or matchup each.
- No tables, no bold, no headers other than ##.
- If a stat is missing from the input, do not fabricate one.
- When a "Conditions" line is supplied (weather, wind, day/night), factor
  it in only where it plausibly matters (wind and HR power, extreme heat
  or cold); otherwise ignore it.
- Use pitcher handedness and batter sides for platoon observations when
  both are supplied.
- When a "Lineups" block is supplied below, the bullets in "Bats to Watch"
  must reference batters by name from the supplied lineup, not paraphrase
  team-level strengths. When the block is absent (no lineup posted), draw
  from the team-line context.`

// RenderPrompt returns the (system, user) message pair to send to the LLM.
func RenderPrompt(c Context) (system, user string) {
	var b strings.Builder

	fmt.Fprintf(&b, "Matchup: %s @ %s\n", labelTeam(c.Away), labelTeam(c.Home))
	if c.GameDateLocal != "" {
		fmt.Fprintf(&b, "First pitch: %s\n", c.GameDateLocal)
	}
	if c.Venue != "" {
		fmt.Fprintf(&b, "Venue: %s\n", c.Venue)
	}
	if cond := formatConditions(c); cond != "" {
		fmt.Fprintf(&b, "Conditions: %s\n", cond)
	}
	fmt.Fprintln(&b)

	writeTeam(&b, "Away", c.Away)
	writeTeam(&b, "Home", c.Home)

	fmt.Fprintln(&b, "Probable starters:")
	writeProbable(&b, "Away", c.Probables[0])
	writeProbable(&b, "Home", c.Probables[1])

	if len(c.Lineups[0].Batters) > 0 || len(c.Lineups[1].Batters) > 0 {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "Lineups (top by batting order):")
		writeLineupBlock(&b, "Away", c.Lineups[0])
		writeLineupBlock(&b, "Home", c.Lineups[1])
	}

	return systemPrompt, strings.TrimRight(b.String(), "\n")
}

func writeLineupBlock(b *strings.Builder, side string, l LineupCtx) {
	fmt.Fprintf(b, "  %s:\n", side)
	limit := topBatters
	if limit > len(l.Batters) {
		limit = len(l.Batters)
	}
	for i := 0; i < limit; i++ {
		x := l.Batters[i]
		pos := x.Position
		if pos == "" {
			pos = "-"
		}
		side := ""
		if x.BatSide != "" {
			side = " (" + x.BatSide + ")"
		}
		if x.SeasonLine != nil {
			fmt.Fprintf(b, "    %d. %s %s%s: AVG %s / OBP %s / OPS %s%s\n",
				x.BattingOrder, x.Name, pos, side,
				x.SeasonLine.AVG, x.SeasonLine.OBP, x.SeasonLine.OPS,
				formatRecentBatting(x.Recent))
		} else if x.Name != "" {
			fmt.Fprintf(b, "    %d. %s %s%s: stats unavailable%s\n",
				x.BattingOrder, x.Name, pos, side, formatRecentBatting(x.Recent))
		}
	}
}

// formatRecentBatting renders a batter's rolling-window line as a
// "; last 7: ..." suffix, or "" when the window fetch came back empty.
func formatRecentBatting(r *api.BatterWindowStats) string {
	if r == nil || r.GamesPlayed == 0 {
		return ""
	}
	return fmt.Sprintf("; last %d games: AVG %s, OPS %s, %d HR",
		r.GamesPlayed, r.AVG, r.OPS, r.HomeRuns)
}

func labelTeam(t TeamCtx) string {
	if t.Abbreviation != "" {
		return t.Abbreviation
	}
	return t.Name
}

func writeTeam(b *strings.Builder, side string, t TeamCtx) {
	fmt.Fprintf(b, "%s: %s (%s)\n", side, t.Name, t.Record)
	if form := formatTeamForm(t); form != "" {
		fmt.Fprintf(b, "  form: %s\n", form)
	}
	if t.SeasonHit != nil {
		fmt.Fprintf(b, "  hitting: AVG %s / OBP %s / SLG %s / OPS %s, HR %d, RBI %d\n",
			t.SeasonHit.AVG, t.SeasonHit.OBP, t.SeasonHit.SLG, t.SeasonHit.OPS,
			t.SeasonHit.HomeRuns, t.SeasonHit.RBI)
	} else {
		fmt.Fprintln(b, "  hitting: stats unavailable")
	}
}

func formatTeamForm(t TeamCtx) string {
	var parts []string
	if t.Streak != "" {
		parts = append(parts, "streak "+t.Streak)
	}
	if t.LastTen != "" {
		parts = append(parts, "last ten "+t.LastTen)
	}
	if t.DivisionRank != "" {
		rank := "division rank " + t.DivisionRank
		if t.GamesBack != "" && t.GamesBack != "-" {
			rank += " (" + t.GamesBack + " GB)"
		}
		parts = append(parts, rank)
	}
	return strings.Join(parts, ", ")
}

// arsenalPitchesShown caps how many pitch types (by usage) go into the
// prompt per starter.
const arsenalPitchesShown = 5

func writeProbable(b *strings.Builder, side string, p ProbableCtx) {
	name := p.Name
	if name == "" {
		name = "TBD"
	}
	if p.HandsThrows != "" {
		name += " (" + p.HandsThrows + ")"
	}
	fmt.Fprintf(b, "  %s: %s\n", side, name)
	if p.SeasonLine != nil {
		fmt.Fprintf(b, "    season: %d-%d, ERA %s, WHIP %s, K/9 %s, IP %s\n",
			p.SeasonLine.Wins, p.SeasonLine.Losses,
			p.SeasonLine.ERA, p.SeasonLine.WHIP, p.SeasonLine.K9, p.SeasonLine.IP)
	} else if p.Name != "" {
		fmt.Fprintln(b, "    season: stats unavailable")
	}
	if len(p.Arsenal) > 0 {
		fmt.Fprintf(b, "    arsenal: %s\n", formatArsenal(p.Arsenal))
	}
	if len(p.RecentStarts) > 0 {
		fmt.Fprintln(b, "    recent starts (most recent first):")
		for _, s := range trimStarts(p.RecentStarts) {
			fmt.Fprintf(b, "      %s vs %s: %s IP, %d ER, %d K, %d BB, %d HR\n",
				s.Date, s.OpponentName, s.InningsPitched,
				s.EarnedRuns, s.Strikeouts, s.Walks, s.HomeRuns)
		}
	}
}

func formatArsenal(pitches []api.ArsenalPitch) string {
	sorted := make([]api.ArsenalPitch, len(pitches))
	copy(sorted, pitches)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].UsagePct > sorted[j].UsagePct })
	if len(sorted) > arsenalPitchesShown {
		sorted = sorted[:arsenalPitchesShown]
	}
	var parts []string
	for _, p := range sorted {
		s := fmt.Sprintf("%s %.0f%%", p.Description, p.UsagePct)
		if p.AvgVelocity > 0 {
			s += fmt.Sprintf(" (%.1f mph)", p.AvgVelocity)
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, ", ")
}

func trimStarts(starts []api.GameLogStart) []api.GameLogStart {
	if len(starts) > recentStartsLimit {
		return starts[:recentStartsLimit]
	}
	return starts
}

// formatConditions merges weather and day/night into one prompt line,
// returning "" when nothing is known yet.
func formatConditions(c Context) string {
	var parts []string
	if c.Weather.Temp != "" {
		parts = append(parts, c.Weather.Temp+"°F")
	}
	if c.Weather.Condition != "" {
		parts = append(parts, c.Weather.Condition)
	}
	if c.Weather.Wind != "" {
		parts = append(parts, "wind "+c.Weather.Wind)
	}
	if c.DayNight != "" {
		parts = append(parts, c.DayNight+" game")
	}
	return strings.Join(parts, ", ")
}
