package config

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestScoutingEnabled(t *testing.T) {
	cases := []struct {
		name string
		s    *Scouting
		want bool
	}{
		{"nil", nil, false},
		{"empty", &Scouting{}, false},
		{"only provider", &Scouting{Provider: "anthropic"}, false},
		{"only key", &Scouting{APIKey: "k"}, false},
		{"missing model", &Scouting{Provider: "anthropic", APIKey: "k"}, false},
		{"all present", &Scouting{Provider: "anthropic", APIKey: "k", Model: "claude"}, true},
	}
	for _, tc := range cases {
		c := &Config{Scouting: tc.s}
		if got := c.ScoutingEnabled(); got != tc.want {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestOldConfigLoadsWithoutScouting(t *testing.T) {
	raw := `{"favorite_teams":["x"],"focus_favorite_team":false}`
	var c Config
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if c.ScoutingEnabled() {
		t.Error("ScoutingEnabled should be false on legacy config")
	}
	if c.Scouting != nil {
		t.Errorf("Scouting should be nil on legacy config, got %+v", c.Scouting)
	}
	if c.ScoutingValue().Provider != "" {
		t.Error("ScoutingValue().Provider should be empty")
	}
}

func TestScoutingOmitemptyOnMarshal(t *testing.T) {
	c := Config{FavoriteTeams: []string{}}
	data, err := json.Marshal(&c)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if strings.Contains(string(data), `"scouting"`) {
		t.Errorf("unset Scouting should be omitempty:\n%s", data)
	}
}

func TestGetKey_ScoutingAPIKeyMasked(t *testing.T) {
	c := &Config{Scouting: &Scouting{APIKey: "sk-ant-1234ABCD"}}
	got, err := c.GetKey("scouting.api_key")
	if err != nil {
		t.Fatalf("GetKey: %v", err)
	}
	if strings.Contains(got, "1234") {
		t.Errorf("masked value leaked prefix: %q", got)
	}
	if !strings.HasSuffix(got, "ABCD") {
		t.Errorf("masked value should preserve last 4 chars, got %q", got)
	}
}

func TestGetKey_ScoutingAPIKeyEmpty(t *testing.T) {
	c := &Config{}
	got, err := c.GetKey("scouting.api_key")
	if err != nil {
		t.Fatalf("GetKey: %v", err)
	}
	if got != "" {
		t.Errorf("empty key should yield empty mask, got %q", got)
	}
}
