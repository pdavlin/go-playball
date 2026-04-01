package buildinfo

import (
	"strings"
	"testing"
)

func TestDefaultVersion(t *testing.T) {
	if v := Version(); v != "dev" {
		t.Errorf("expected default version 'dev', got %q", v)
	}
}

func TestStringFormat(t *testing.T) {
	s := String()
	if !strings.HasPrefix(s, "go-playball ") {
		t.Errorf("expected prefix 'go-playball ', got %q", s)
	}
	if !strings.Contains(s, "commit:") {
		t.Errorf("expected 'commit:' in output, got %q", s)
	}
	if !strings.Contains(s, "built:") {
		t.Errorf("expected 'built:' in output, got %q", s)
	}
}
