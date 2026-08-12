package scouting

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pdavlin/go-playball/internal/api"
	"github.com/pdavlin/go-playball/internal/savant"
)

// scoutingPreamble opens the pregame system prompt: role, the three required
// section headers, the appended-section license, and the verified-spine
// framing.
const scoutingPreamble = `You are a major-league baseball scout writing a tight,
pre-game scouting report for a fan who has thirty seconds. Output Markdown
with these literal section headers, in order:

## The Edge
## Pitching Edge
## Bats to Watch

After those three, you are encouraged to append additional ## sections
of your choosing at the bottom whenever the supplied facts contain
something material a fan should know: weather or wind that changes run
scoring, a team riding a hot or cold streak, a batter whose recent form
diverges sharply from his season line, a starter trending up or down
across his last few outings, park quirks, or anything else concrete.
Examples: ## Weather, ## Hot bats, ## Trending, ## X-factor. Add as many
as the facts support, but every extra section must be grounded in a stat
or condition from the input — do not pad with generic filler.

The facts below the instructions are the verified spine of this report:
the app computed them from the MLB stats feed, and they are the only
source you may draw on. Treat them as authoritative and narrate only what
they contain.`

// sharedRules is the grounding, Savant, and voice guidance common to every
// scouting tense. Both the pregame and in-progress system prompts end with it
// so the rules never drift between tenses.
const sharedRules = `Grounding rules:
- Every number you write — records, ERAs, WHIPs, K/9, innings, AVG, OBP,
  SLG, OPS, home runs, RBI, pitch velocities, dates — must appear verbatim
  in the facts. Do not compute a new number, average, gap, or record of
  your own.
- If a number or player is not in the facts, do not cite it. When unsure,
  omit the number rather than estimate it.
- The recent-start lines are context on a starter's current form, not
  events in tonight's game, and they are NOT a career record against
  tonight's opponent. Never write "[Pitcher] is N-M vs [Team]" — the facts
  carry no career-vs-opponent record, so stating one is an invention.
- Do not invent bullpen availability, injuries, roster moves, or head-to-
  head history; none of those are supplied.
- Expected and percentile stats appear ONLY when a "savant pct rank" line
  is supplied for that player. Where such a line is absent, treat
  expected/percentile stats (xERA, xwOBA, xBA, and the like) as unavailable
  and do not cite or estimate them.

Expected vs actual (Baseball Savant percentiles):
- A "savant pct rank" line gives Baseball Savant percentile ranks (0-100)
  where a HIGHER percentile is always better for that player, including for
  K, whiff, and chase, which Savant already inverts. These are ranks, not
  the raw rates.
- When both an expected rank and its actual counterpart are supplied for a
  player — xwOBA vs wOBA, or xSLG vs SLG for a hitter, xERA for a pitcher
  against his season ERA — make the GAP the story where one clearly exists.
  A hitter whose xwOBA rank sits well above his wOBA rank has hit into bad
  luck and is a bounce-back bet; the reverse flags a hot streak likely to
  cool. A starter with an elite xERA rank beneath a plain season ERA is
  pitching better than his line.
- Describe the gap qualitatively (expected ahead of actual, or behind).
  Never state a numeric gap the facts do not contain: cite each supplied
  rank as given and do not subtract one from another to report a new
  number.

Voice rules:
- Write every appended section title and all prose in sentence case, never
  Title Case: "## Hot bats", not "## Hot Bats".
- Do not start a line with a fact label ("Away:", "Probable starters:",
  "form:", "arsenal:", "recent starts:"). Those are input anchors for you
  to parse, not prose.
- Avoid hype words (dramatic, clutch, huge, stunning, explosive, thrilling,
  historic, improbable) and betting references (picks, odds, spreads,
  lines). Grounded analytical adjectives like "tight", "lopsided",
  "punishing", or "emphatic" are fine.
- Each bullet covers a distinct angle. Do not repeat a player, storyline,
  or phrasing between bullets.`

// pregameConstraints closes the pregame system prompt with the per-section
// format rules specific to the preview tense.
const pregameConstraints = `Constraints:
- "## The Edge" must be exactly one sentence naming who holds the
  analytical edge and why. State an EDGE, never a win prediction,
  probability, betting line, or pick.
- "## Pitching Edge" contrasts the two starters in 2-4 short lines: season
  lines, recent form, and velocity or pitch mix where the arsenal is given.
- "## Bats to Watch" lists 2-4 bullets, one batter or matchup each.
- No tables, no bold, no headers other than ##.
- When a "Conditions" line is supplied (weather, wind, day/night), factor
  it in only where it plausibly matters (wind and HR power, extreme heat
  or cold); otherwise ignore it.
- Use pitcher handedness and batter sides for platoon observations when
  both are supplied.
- When a "Lineups" block is supplied below, the bullets in "Bats to Watch"
  must reference batters by name from the supplied lineup, not paraphrase
  team-level strengths. When the block is absent (no lineup posted), draw
  from the team-line context.`

// systemPrompt is the pregame preview system prompt, composed from the shared
// rule fragments so grounding and voice stay identical across tenses.
const systemPrompt = scoutingPreamble + "\n\n" + sharedRules + "\n\n" + pregameConstraints

// inProgressPreamble opens the in-progress system prompt. It sets the present
// tense, forbids play-by-play and invented scoring events, tells the model not
// to restate the score the app already shows, and lays out the live sections.
const inProgressPreamble = `You are a major-league baseball scout writing a tight,
in-progress read on a game already under way, for a fan who has thirty
seconds. This game is IN PROGRESS, so write in the present tense. Output
Markdown with these literal section headers, in order:

## The Duel
## Hitter Watch
## What to Watch

You may append additional ## sections at the bottom when a supplied fact
supports one: a bullpen or availability angle, a standings or series stake,
weather, or a park effect. Examples: ## Bullpen, ## Standings, ## Weather.
Every extra section must be grounded in a fact — do not pad with filler.

The app already shows the fan the current score, inning, count, and
base-out state next to your text. Treat the "Live game state" facts as
context to reason from — do not restate the score, the inning, or the
base-out state as prose, and do not write a score line.

Absolute prohibitions:
- Never write a play-by-play sentence. Do not narrate the sequence of a
  plate appearance, an at-bat, or an inning ("singles, then scores",
  "grounds out to end the frame"). Give analytical context, not a call of
  the game.
- Never reference a scoring event that is not in the facts. Which batter
  drove in which run is not supplied — do not invent it.
- If no runs have scored yet, frame it as a scoreless duel and do not imply
  a lead either way.

The facts below the instructions are the verified spine of this report:
the app computed them from the MLB stats feed, and they are the only
source you may draw on. Treat them as authoritative and narrate only what
they contain.

Section guidance:
- "## The Duel" contrasts the two starters: their season ERAs and recent
  form, then how each has looked so far from the "so far" line. When a
  starter's so-far line is already his final line (he has been pulled),
  read it as his outing for the night.
- "## Hitter Watch" names a strong season hitter from each side who has not
  broken through in this game yet — a bat still due to surface.
- "## What to Watch" covers a bullpen or strategic angle, a standings or
  series implication, or a weather effect — only where a supplied fact
  backs it. Skip it rather than pad.`

// inProgressSystemPrompt is the in-progress system prompt, sharing the same
// grounding and voice rules as the pregame prompt.
const inProgressSystemPrompt = inProgressPreamble + "\n\n" + sharedRules

// RenderPrompt returns the (system, user) message pair to send to the LLM.
// The tense is selected by c.State: an in-progress context swaps in the
// in-progress system prompt and appends the live-state fact block.
func RenderPrompt(c Context) (system, user string) {
	var b strings.Builder

	writeSpine(&b, c)
	if c.State == StateInProgress {
		writeLiveBlock(&b, c.Live)
	}

	user = strings.TrimRight(b.String(), "\n")
	if c.State == StateInProgress {
		return inProgressSystemPrompt, user
	}
	return systemPrompt, user
}

// writeSpine writes the matchup header, team lines, probable starters, and
// lineups — the fact spine shared by every scouting tense.
func writeSpine(b *strings.Builder, c Context) {
	fmt.Fprintf(b, "Matchup: %s @ %s\n", labelTeam(c.Away), labelTeam(c.Home))
	if c.GameDateLocal != "" {
		fmt.Fprintf(b, "First pitch: %s\n", c.GameDateLocal)
	}
	if c.Venue != "" {
		fmt.Fprintf(b, "Venue: %s\n", c.Venue)
	}
	if cond := formatConditions(c); cond != "" {
		fmt.Fprintf(b, "Conditions: %s\n", cond)
	}
	fmt.Fprintln(b)

	writeTeam(b, "Away", c.Away)
	writeTeam(b, "Home", c.Home)

	fmt.Fprintln(b, "Probable starters:")
	writeProbable(b, "Away", c.Probables[0])
	writeProbable(b, "Home", c.Probables[1])

	if len(c.Lineups[0].Batters) > 0 || len(c.Lineups[1].Batters) > 0 {
		fmt.Fprintln(b)
		fmt.Fprintln(b, "Lineups (top by batting order):")
		writeLineupBlock(b, "Away", c.Lineups[0])
		writeLineupBlock(b, "Home", c.Lineups[1])
	}
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
		if x.Name != "" {
			if line := formatBatterSavant(x.XStats); line != "" {
				fmt.Fprintf(b, "       %s\n", line)
			}
		}
	}
}

// formatBatterSavant renders a hitter's Savant percentile ranks as a
// "savant pct rank: ..." line, or "" when no ranks are supplied. Nil
// fields are dropped so a low-sample hitter shows only what Savant has.
func formatBatterSavant(p *savant.Percentiles) string {
	if p == nil {
		return ""
	}
	parts := joinSavantParts(
		pctPart("xwOBA", p.XWOBAPct),
		pctPart("wOBA", p.WOBAPct),
		pctPart("xBA", p.XBAPct),
		pctPart("xSLG", p.XSLGPct),
		pctPart("SLG", p.SLGPct),
		pctPart("hard-hit", p.HardHitPct),
		pctPart("barrel", p.BarrelPct),
		pctPart("K", p.KPct),
		pctPart("whiff", p.WhiffPct),
		pctPart("chase", p.ChasePct),
		pctPart("sprint", p.SprintSpeedPct),
	)
	if parts == "" {
		return ""
	}
	return "savant pct rank: " + parts
}

// formatPitcherSavant renders a starter's Savant percentile ranks. The
// expected/actual pair here is xERA against the season ERA already on the
// season line, plus the stuff and contact-suppression ranks.
func formatPitcherSavant(p *savant.Percentiles) string {
	if p == nil {
		return ""
	}
	parts := joinSavantParts(
		pctPart("xERA", p.XERAPct),
		pctPart("xwOBA", p.XWOBAPct),
		pctPart("K", p.KPct),
		pctPart("BB", p.BBPct),
		pctPart("whiff", p.WhiffPct),
		pctPart("chase", p.ChasePct),
		pctPart("fastball velo", p.FastballVeloPct),
		pctPart("barrel", p.BarrelPct),
		pctPart("hard-hit", p.HardHitPct),
	)
	if parts == "" {
		return ""
	}
	return "savant pct rank: " + parts
}

// pctPart formats one "label N" percentile fragment, or "" when the rank
// is nil (player carried a null for that metric).
func pctPart(label string, v *int) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%s %d", label, *v)
}

// joinSavantParts joins the non-empty fragments with ", ".
func joinSavantParts(parts ...string) string {
	kept := parts[:0]
	for _, p := range parts {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, ", ")
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
	if line := formatPitcherSavant(p.XStats); line != "" {
		fmt.Fprintf(b, "    %s\n", line)
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
