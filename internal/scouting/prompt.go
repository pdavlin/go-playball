package scouting

import (
	"fmt"
	"strings"
)

const systemPrompt = `You are a major-league baseball scout writing a tight,
pre-game scouting report for a fan who has thirty seconds. Output Markdown
with these literal section headers, in order:

## The Edge
## Pitching Edge
## Bats to Watch

After those three, you may add at most two additional ## sections of your
choosing (e.g. ## Bullpen Notes, ## Weather, ## X-Factor) when you have
something concrete to say. Do not invent sections without content.

Constraints:
- "## The Edge" must be exactly one sentence naming who has the advantage
  and why.
- "## Pitching Edge" compares the two starters in 2-4 short lines.
- "## Bats to Watch" lists 2-4 bullets, one batter or matchup each.
- No tables, no bold, no headers other than ##.
- If a stat is missing from the input, do not fabricate one.
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
		if x.SeasonLine != nil {
			fmt.Fprintf(b, "    %d. %s %s: AVG %s / OBP %s / OPS %s\n",
				x.BattingOrder, x.Name, pos,
				x.SeasonLine.AVG, x.SeasonLine.OBP, x.SeasonLine.OPS)
		} else if x.Name != "" {
			fmt.Fprintf(b, "    %d. %s %s: stats unavailable\n",
				x.BattingOrder, x.Name, pos)
		}
	}
}

func labelTeam(t TeamCtx) string {
	if t.Abbreviation != "" {
		return t.Abbreviation
	}
	return t.Name
}

func writeTeam(b *strings.Builder, side string, t TeamCtx) {
	fmt.Fprintf(b, "%s: %s (%s)\n", side, t.Name, t.Record)
	if t.SeasonHit != nil {
		fmt.Fprintf(b, "  hitting: AVG %s / OBP %s / SLG %s / OPS %s, HR %d, RBI %d\n",
			t.SeasonHit.AVG, t.SeasonHit.OBP, t.SeasonHit.SLG, t.SeasonHit.OPS,
			t.SeasonHit.HomeRuns, t.SeasonHit.RBI)
	} else {
		fmt.Fprintln(b, "  hitting: stats unavailable")
	}
}

func writeProbable(b *strings.Builder, side string, p ProbableCtx) {
	name := p.Name
	if name == "" {
		name = "TBD"
	}
	fmt.Fprintf(b, "  %s: %s\n", side, name)
	if p.SeasonLine != nil {
		fmt.Fprintf(b, "    season: %d-%d, ERA %s, WHIP %s, K/9 %s, IP %s\n",
			p.SeasonLine.Wins, p.SeasonLine.Losses,
			p.SeasonLine.ERA, p.SeasonLine.WHIP, p.SeasonLine.K9, p.SeasonLine.IP)
	} else if p.Name != "" {
		fmt.Fprintln(b, "    season: stats unavailable")
	}
}
