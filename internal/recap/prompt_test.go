package recap

import (
	"strings"
	"testing"
)

func TestSystemPrompt_HasRequiredHeaders(t *testing.T) {
	system, _ := RenderPrompt(Context{})
	want := []string{
		"## How It Was Won",
		"## Turning Point",
		"## On the Mound",
		"## Top Performer",
		"## Bullpen",
	}
	for _, h := range want {
		if !strings.Contains(system, h) {
			t.Errorf("system prompt missing header %q", h)
		}
	}
}

func TestSystemPrompt_PastTenseAndNoScoreRestatement(t *testing.T) {
	system, _ := RenderPrompt(Context{})
	if !strings.Contains(system, "past tense") {
		t.Error("recap system prompt should require past tense throughout")
	}
	if !strings.Contains(system, "DO NOT restate the score") {
		t.Error("recap system prompt should tell the model not to restate the score")
	}
	if !strings.Contains(system, "do not write a score line") {
		t.Error("recap system prompt should forbid a score line")
	}
}

func TestSystemPrompt_AlignedSectionGuidance(t *testing.T) {
	system, _ := RenderPrompt(Context{})
	// Decisive inning + standout hitter, winning pitcher line, and the
	// standings/series implication must all be reflected in the guidance.
	for _, phrase := range []string{
		"winning pitcher's line (IP/H/R/ER/K)",
		"their game line (H/AB, HR, RBI)",
		"a move in the standings",
	} {
		if !strings.Contains(system, phrase) {
			t.Errorf("recap system prompt missing aligned guidance %q", phrase)
		}
	}
}

func TestSystemPrompt_GroundingAndVoiceRules(t *testing.T) {
	system, _ := RenderPrompt(Context{})
	if !strings.Contains(system, "must appear verbatim") {
		t.Error("recap system prompt should require numbers verbatim from the facts")
	}
	if !strings.Contains(system, "Do not invent a season record") {
		t.Error("recap system prompt should forbid invented records")
	}
	if !strings.Contains(system, "sentence case") {
		t.Error("recap system prompt should require sentence-case prose")
	}
	if !strings.Contains(system, "Avoid hype words") {
		t.Error("recap system prompt should carry the hype-word avoidance rule")
	}
	if !strings.Contains(system, "distinct angle") {
		t.Error("recap system prompt should require a distinct angle per section")
	}
}

func TestRenderPrompt_FinalLineAndDecisions(t *testing.T) {
	ctx := Context{
		Away: TeamScore{Name: "Boston Red Sox", Abbreviation: "BOS", Runs: 5, Hits: 9},
		Home: TeamScore{Name: "New York Yankees", Abbreviation: "NYY", Runs: 3, Hits: 7, Errors: 1},
		Decisions: DecisionsCtx{
			Winner: &PitcherDecision{Name: "Jane Winner"},
			Loser:  &PitcherDecision{Name: "Bob Loser"},
		},
	}
	_, user := RenderPrompt(ctx)
	if !strings.Contains(user, "Game: Boston Red Sox (BOS) at New York Yankees (NYY)") {
		t.Errorf("missing game header: %s", user)
	}
	if !strings.Contains(user, "Final: Red Sox 5, Yankees 3") {
		t.Errorf("missing final line with nicknames: %s", user)
	}
	if !strings.Contains(user, "W: Jane Winner") {
		t.Errorf("missing winner line: %s", user)
	}
	if !strings.Contains(user, "L: Bob Loser") {
		t.Errorf("missing loser line: %s", user)
	}
	if !strings.Contains(user, "SV: none recorded") {
		t.Errorf("missing 'none recorded' for empty SV: %s", user)
	}
}

func TestSystemPrompt_DoesNotRepeatBannedPhrases(t *testing.T) {
	// Telling the model "do not use X" plants X. Verify the system
	// prompt does NOT contain the canonical abstract framings, so the
	// model isn't primed to echo them.
	system, _ := RenderPrompt(Context{})
	for _, phrase := range []string{
		"the visitors",
		"the away team",
		"the home team",
		"the hosts",
	} {
		if strings.Contains(system, phrase) {
			t.Errorf("system prompt should not echo %q (negation backfires)", phrase)
		}
	}
}

func TestRenderPrompt_Standouts(t *testing.T) {
	ctx := Context{
		Away: TeamScore{Name: "Arizona Diamondbacks", Abbreviation: "ARI", Runs: 5},
		Home: TeamScore{Name: "San Francisco Giants", Abbreviation: "SF", Runs: 3},
		Standouts: [2]TeamStandouts{
			{
				Hitters: []HitterLine{
					{Name: "Three Hits", AB: 4, H: 3, HR: 1, RBI: 2},
				},
				Pitchers: []PitcherLine{
					{Name: "Jane Winner", IP: "7.0", K: 8, BB: 1, ER: 1, Decision: "W"},
				},
			},
			{
				Hitters: nil,
				Pitchers: []PitcherLine{
					{Name: "Sam Saver", IP: "1.0", K: 1, BB: 0, ER: 0, Decision: "SV"},
				},
			},
		},
	}
	_, user := RenderPrompt(ctx)
	if !strings.Contains(user, "H Three Hits: 3-for-4, 1 HR, 2 RBI") {
		t.Errorf("hitter line wrong: %s", user)
	}
	if !strings.Contains(user, "P Jane Winner (W): 7.0 IP, 8 K, 1 BB, 1 ER") {
		t.Errorf("pitcher line wrong: %s", user)
	}
	if !strings.Contains(user, "Diamondbacks standouts:") {
		t.Errorf("missing away standouts header (nickname): %s", user)
	}
	if !strings.Contains(user, "Giants standouts:") {
		t.Errorf("missing home standouts header (nickname): %s", user)
	}
	if !strings.Contains(user, "P Sam Saver (SV)") {
		t.Errorf("missing save tag: %s", user)
	}
}

func TestRenderPrompt_EmptyStandoutsLineNotRecorded(t *testing.T) {
	ctx := Context{
		Away: TeamScore{Name: "Cleveland Guardians", Abbreviation: "CLE"},
		Home: TeamScore{Name: "Washington Nationals", Abbreviation: "WSH"},
	}
	_, user := RenderPrompt(ctx)
	if !strings.Contains(user, "Guardians standouts:\n  (none recorded)") {
		t.Errorf("missing none-recorded line:\n%s", user)
	}
}

func TestTeamNickname(t *testing.T) {
	cases := map[string]string{
		"Arizona Diamondbacks":   "Diamondbacks",
		"San Francisco Giants":   "Giants",
		"Boston Red Sox":         "Red Sox",
		"Chicago White Sox":      "White Sox",
		"Toronto Blue Jays":      "Blue Jays",
		"New York Yankees":       "Yankees",
		"St. Louis Cardinals":    "Cardinals",
	}
	for name, want := range cases {
		got := teamNickname(TeamScore{Name: name})
		if got != want {
			t.Errorf("teamNickname(%q) = %q, want %q", name, got, want)
		}
	}
	if got := teamNickname(TeamScore{Abbreviation: "XYZ"}); got != "XYZ" {
		t.Errorf("name-empty fallback to abbreviation failed: %q", got)
	}
	// Explicit Nickname field beats name-extraction.
	if got := teamNickname(TeamScore{Name: "Cincinnati Reds", Nickname: "Reds"}); got != "Reds" {
		t.Errorf("explicit Nickname should win: %q", got)
	}
	if got := teamNickname(TeamScore{Name: "Should Be Ignored", Nickname: "Custom"}); got != "Custom" {
		t.Errorf("explicit Nickname should override extraction: %q", got)
	}
}
