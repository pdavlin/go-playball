package savant

import "testing"

// syntheticPage mimics the shape of a real Savant percentile-rankings
// page: a small `var leaderboard_data = [ ... ];` assignment surrounded by
// unrelated markup and script. Two entries — one fully populated, one
// carrying nulls for low sample — exercise the null handling and the
// player-id keying without shipping the 900KB real fixture.
const syntheticPage = `<!DOCTYPE html><html><head><title>Percentile Rankings</title></head>
<body><div id="leaderboard"></div>
<script>
var somethingElse = [1, 2, 3];
var leaderboard_data = [
  {"player_name":"Keith, Colt","team_name":"Tigers","year":"2026","player_type":"batter","player_id":"690993","percent_rank_xwoba":59,"percent_rank_woba":46,"percent_rank_xba":81,"percent_rank_xslg":65,"percent_rank_slg":49,"percent_rank_hard_hit_percent":48,"percent_rank_barrel_batted_rate":41,"percent_rank_k_percent":66,"percent_rank_bb_percent":17,"percent_rank_whiff_percent":71,"percent_rank_chase_percent":81,"percent_speed_order":70,"href":"<a href=\"/savant-player/690993\">Keith, Colt</a>"},
  {"player_name":"Simon, Ronny","team_name":"Pirates","year":"2026","player_type":"batter","player_id":"682927","percent_rank_xwoba":null,"percent_rank_woba":null,"percent_rank_xba":null,"percent_rank_hard_hit_percent":null,"percent_rank_k_percent":null,"percent_speed_order":81,"href":"<a href=\"/savant-player/682927\">Simon, Ronny</a>"}
];
var moreScript = true;
</script>
</body></html>`

func TestParseLeaderboardData_KeysByPlayerID(t *testing.T) {
	players, err := ParseLeaderboardData(syntheticPage)
	if err != nil {
		t.Fatalf("ParseLeaderboardData: %v", err)
	}
	if len(players) != 2 {
		t.Fatalf("expected 2 players, got %d", len(players))
	}
	if _, ok := players["690993"]; !ok {
		t.Errorf("map missing key for populated player 690993")
	}
	if _, ok := players["682927"]; !ok {
		t.Errorf("map missing key for null-heavy player 682927")
	}
}

func TestParseLeaderboardData_PopulatedFields(t *testing.T) {
	players, err := ParseLeaderboardData(syntheticPage)
	if err != nil {
		t.Fatalf("ParseLeaderboardData: %v", err)
	}
	keith := players["690993"]
	if keith.Name != "Keith, Colt" {
		t.Errorf("player_name = %q, want %q", keith.Name, "Keith, Colt")
	}
	if keith.XWOBAPct == nil || *keith.XWOBAPct != 59 {
		t.Errorf("XWOBAPct = %v, want 59", keith.XWOBAPct)
	}
	if keith.WOBAPct == nil || *keith.WOBAPct != 46 {
		t.Errorf("WOBAPct = %v, want 46", keith.WOBAPct)
	}
	if keith.ChasePct == nil || *keith.ChasePct != 81 {
		t.Errorf("ChasePct = %v, want 81", keith.ChasePct)
	}
	if keith.SprintSpeedPct == nil || *keith.SprintSpeedPct != 70 {
		t.Errorf("SprintSpeedPct = %v, want 70", keith.SprintSpeedPct)
	}
}

func TestParseLeaderboardData_NullsBecomeNil(t *testing.T) {
	players, err := ParseLeaderboardData(syntheticPage)
	if err != nil {
		t.Fatalf("ParseLeaderboardData: %v", err)
	}
	simon := players["682927"]
	if simon.XWOBAPct != nil {
		t.Errorf("null XWOBAPct should decode to nil, got %v", *simon.XWOBAPct)
	}
	if simon.KPct != nil {
		t.Errorf("null KPct should decode to nil, got %v", *simon.KPct)
	}
	// A non-null field on the same low-sample entry still populates.
	if simon.SprintSpeedPct == nil || *simon.SprintSpeedPct != 81 {
		t.Errorf("SprintSpeedPct = %v, want 81", simon.SprintSpeedPct)
	}
}

func TestParseLeaderboardData_MissingArray(t *testing.T) {
	_, err := ParseLeaderboardData("<html><body>no data here</body></html>")
	if err == nil {
		t.Error("expected an error when leaderboard_data is absent")
	}
}
