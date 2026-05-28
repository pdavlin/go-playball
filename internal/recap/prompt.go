package recap

import (
	"fmt"
	"strings"
)

const systemPrompt = `You are a major-league baseball recap writer for a fan
who missed the game and has thirty seconds. Output Markdown with exactly
these five section headers, in order, each followed by 1-2 sentences of
prose. No bullets, no tables, no bold, no other headers.

## How It Was Won
## Turning Point
## On the Mound
## Top Performer
## Bullpen

Section guidance:
- "## How It Was Won" frames the causal arc of the game in one sentence
  and makes the result unambiguous (winning team and final score). What
  the winning team did, or what the losing team failed to do.
- "## Turning Point" identifies one inning or one play that decided the
  game and explains in 1-2 sentences why. Even in a blowout, name the
  moment the result became inevitable.
- "## On the Mound" is about the starters - typically the losing
  starter's struggle or the winning starter's command. One or both.
- "## Top Performer" names one hitter from the supplied Standouts block
  and their line. Pick the most impactful bat regardless of team.
- "## Bullpen" is about relief work - depth, pivotal outs, a closer's
  save, or a reliever who blew it open. Skip generic praise.

Constraints:
- Reference players ONLY by names that appear in the supplied data
  (Decisions, Standouts). Do not invent names or stat lines.
- Refer to each team by the name written in the data block — either
  the city ("Cleveland", "Washington") or the nickname ("the
  Guardians", "the Nationals"). The data block uses these names in
  every label; mirror them in the prose, in every section.
- If a stat is missing from the input, do not fabricate one.`

// RenderPrompt returns the (system, user) pair to send to the LLM.
func RenderPrompt(c Context) (system, user string) {
	var b strings.Builder

	awayNick := teamNickname(c.Away)
	homeNick := teamNickname(c.Home)

	fmt.Fprintf(&b, "Game: %s at %s\n", teamFullLine(c.Away), teamFullLine(c.Home))
	if c.Venue != "" {
		fmt.Fprintf(&b, "Venue: %s\n", c.Venue)
	}
	fmt.Fprintf(&b, "Final: %s %d, %s %d\n", awayNick, c.Away.Runs, homeNick, c.Home.Runs)
	fmt.Fprintln(&b)

	writeDecisions(&b, c.Decisions)
	writeLinescore(&b, awayNick, homeNick, c)
	writeScoring(&b, awayNick, homeNick, c.Scoring)
	writeStandouts(&b, awayNick, c.Standouts[0])
	writeStandouts(&b, homeNick, c.Standouts[1])

	return systemPrompt, strings.TrimRight(b.String(), "\n")
}

// teamFullLine renders "<Full Name> (<ABBR>)" when both are present, or
// just whichever is non-empty.
func teamFullLine(t TeamScore) string {
	switch {
	case t.Name != "" && t.Abbreviation != "":
		return fmt.Sprintf("%s (%s)", t.Name, t.Abbreviation)
	case t.Name != "":
		return t.Name
	default:
		return t.Abbreviation
	}
}

// teamNickname returns the natural prose handle for a team. Prefers the
// MLB API's authoritative teamName ("Reds", "Red Sox"), then falls back
// to extracting the last word of the full name with special-cases for
// two-word nicknames, then to the abbreviation, then to whatever's left.
// Used in every label in the recap user prompt so the LLM has team
// names to mirror — not generic home/away role labels.
func teamNickname(t TeamScore) string {
	if t.Nickname != "" {
		return t.Nickname
	}
	if t.Name == "" {
		return t.Abbreviation
	}
	for _, suffix := range []string{"Red Sox", "White Sox", "Blue Jays"} {
		if strings.HasSuffix(t.Name, suffix) {
			return suffix
		}
	}
	parts := strings.Fields(t.Name)
	if len(parts) == 0 {
		return t.Name
	}
	return parts[len(parts)-1]
}

func writeDecisions(b *strings.Builder, d DecisionsCtx) {
	fmt.Fprintln(b, "Decisions:")
	fmt.Fprintf(b, "  W: %s\n", decisionName(d.Winner))
	fmt.Fprintf(b, "  L: %s\n", decisionName(d.Loser))
	fmt.Fprintf(b, "  SV: %s\n", decisionName(d.Save))
	fmt.Fprintln(b)
}

func decisionName(p *PitcherDecision) string {
	if p == nil || p.Name == "" {
		return "none recorded"
	}
	return p.Name
}

func writeLinescore(b *strings.Builder, away, home string, c Context) {
	if len(c.Linescore) == 0 {
		return
	}
	fmt.Fprintf(b, "Linescore (%s / %s by inning):\n", away, home)
	for _, ip := range c.Linescore {
		awayCell := "-"
		homeCell := "-"
		if ip.AwayPlayed {
			awayCell = fmt.Sprintf("%d", ip.Away)
		}
		if ip.HomePlayed {
			homeCell = fmt.Sprintf("%d", ip.Home)
		}
		fmt.Fprintf(b, "  %d: %s/%s\n", ip.Inning, awayCell, homeCell)
	}
	fmt.Fprintf(b, "  R/H/E: %s %d/%d/%d  %s %d/%d/%d\n",
		away, c.Away.Runs, c.Away.Hits, c.Away.Errors,
		home, c.Home.Runs, c.Home.Hits, c.Home.Errors)
	fmt.Fprintln(b)
}

func writeScoring(b *strings.Builder, away, home string, plays []ScoringPlay) {
	if len(plays) == 0 {
		return
	}
	fmt.Fprintln(b, "Scoring plays:")
	for _, p := range plays {
		batting := away
		if p.HalfInning == "bottom" {
			batting = home
		}
		fmt.Fprintf(b, "  Inning %d (%s batting): %s [score: %s %d, %s %d]\n",
			p.Inning, batting, p.Description,
			away, p.AwayScore, home, p.HomeScore)
	}
	fmt.Fprintln(b)
}

func writeStandouts(b *strings.Builder, side string, s TeamStandouts) {
	fmt.Fprintf(b, "%s standouts:\n", side)
	if len(s.Hitters) == 0 && len(s.Pitchers) == 0 {
		fmt.Fprintln(b, "  (none recorded)")
		fmt.Fprintln(b)
		return
	}
	for _, h := range s.Hitters {
		fmt.Fprintf(b, "  H %s: %d-for-%d", h.Name, h.H, h.AB)
		if h.HR > 0 {
			fmt.Fprintf(b, ", %d HR", h.HR)
		}
		if h.RBI > 0 {
			fmt.Fprintf(b, ", %d RBI", h.RBI)
		}
		fmt.Fprintln(b)
	}
	for _, p := range s.Pitchers {
		tag := ""
		if p.Decision != "" {
			tag = fmt.Sprintf(" (%s)", p.Decision)
		}
		fmt.Fprintf(b, "  P %s%s: %s IP, %d K, %d BB, %d ER\n",
			p.Name, tag, p.IP, p.K, p.BB, p.ER)
	}
	fmt.Fprintln(b)
}
