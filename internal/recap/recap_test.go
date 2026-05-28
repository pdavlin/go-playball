package recap

import "testing"

func TestPromptHash_Stable(t *testing.T) {
	a := promptHash("sys", "user", "model")
	b := promptHash("sys", "user", "model")
	if a != b {
		t.Errorf("hash unstable: %s vs %s", a, b)
	}
}

func TestPromptHash_ChangesOnInputChange(t *testing.T) {
	base := promptHash("sys", "user", "model")
	cases := map[string]string{
		"system changed": promptHash("sys2", "user", "model"),
		"user changed":   promptHash("sys", "user2", "model"),
		"model changed":  promptHash("sys", "user", "model2"),
	}
	for name, got := range cases {
		if got == base {
			t.Errorf("%s: hash should differ from base %s", name, base)
		}
	}
}
