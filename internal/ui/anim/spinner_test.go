package anim

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestNewSpinner_FrameCount(t *testing.T) {
	s := NewSpinner(10, "Test", red, blue)
	if len(s.frames) != prerenderedFrames {
		t.Fatalf("expected %d frames, got %d", prerenderedFrames, len(s.frames))
	}
	if len(s.dotFrames) != SpinnerFPS {
		t.Fatalf("expected %d dot frames, got %d", SpinnerFPS, len(s.dotFrames))
	}
}

func TestNewSpinner_FrameVisibleWidth(t *testing.T) {
	size := 10
	s := NewSpinner(size, "Test", red, blue)
	for i, frame := range s.frames {
		w := lipgloss.Width(frame)
		if w != size {
			t.Errorf("frame %d: expected visible width %d, got %d", i, size, w)
		}
	}
}

func TestNewSpinner_BirthOffsets(t *testing.T) {
	s := NewSpinner(10, "Test", red, blue)
	if len(s.birthOffsets) != 10 {
		t.Fatalf("expected 10 birth offsets, got %d", len(s.birthOffsets))
	}
	for i, offset := range s.birthOffsets {
		if offset < 0 || offset >= SpinnerFPS {
			t.Errorf("birth offset %d out of range [0, %d): %d", i, SpinnerFPS, offset)
		}
	}
}

func TestSpinner_StartSetsRunning(t *testing.T) {
	s := NewSpinner(5, "Test", red, blue)
	if s.State() != SpinnerPaused {
		t.Fatal("new spinner should be paused")
	}
	s, cmd := s.Start()
	if s.State() != SpinnerRunning {
		t.Fatal("started spinner should be running")
	}
	if cmd == nil {
		t.Fatal("Start should return a tick cmd")
	}
}

func TestSpinner_PauseStopsTicks(t *testing.T) {
	s := NewSpinner(5, "Test", red, blue)
	s, _ = s.Start()
	s = s.Pause()
	if s.State() != SpinnerPaused {
		t.Fatal("paused spinner should report paused state")
	}
	// Update with matching tick should return nil cmd when paused
	s, cmd := s.Update(SpinnerTickMsg{ID: s.id})
	if cmd != nil {
		t.Fatal("update on paused spinner should return nil cmd")
	}
}

func TestSpinner_UpdateAdvancesFrame(t *testing.T) {
	s := NewSpinner(5, "Test", red, blue)
	s, _ = s.Start()
	initialFrame := s.frameIdx
	s, cmd := s.Update(SpinnerTickMsg{ID: s.id})
	if s.frameIdx != initialFrame+1 {
		t.Errorf("expected frameIdx %d, got %d", initialFrame+1, s.frameIdx)
	}
	if cmd == nil {
		t.Fatal("update on running spinner should return next tick cmd")
	}
}

func TestSpinner_UpdateIgnoresWrongID(t *testing.T) {
	s := NewSpinner(5, "Test", red, blue)
	s, _ = s.Start()
	initialFrame := s.frameIdx
	s, cmd := s.Update(SpinnerTickMsg{ID: s.id + 999})
	if s.frameIdx != initialFrame {
		t.Error("update with wrong ID should not advance frame")
	}
	if cmd != nil {
		t.Fatal("update with wrong ID should return nil cmd")
	}
}

func TestSpinner_ViewNonEmpty(t *testing.T) {
	s := NewSpinner(5, "Test", red, blue)
	s, _ = s.Start()
	view := s.View()
	if view == "" {
		t.Fatal("View should return non-empty string when running")
	}
}

func TestSpinner_ViewContainsLabel(t *testing.T) {
	s := NewSpinner(5, "Loading", red, blue)
	s, _ = s.Start()
	view := s.View()
	if !containsStr(view, "Loading") {
		t.Error("View should contain the label text")
	}
}

func TestSpinner_DotFrameVisibleWidth(t *testing.T) {
	size := 10
	s := NewSpinner(size, "Test", red, blue)
	for i, frame := range s.dotFrames {
		w := lipgloss.Width(frame)
		if w != size {
			t.Errorf("dot frame %d: expected visible width %d, got %d", i, size, w)
		}
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
