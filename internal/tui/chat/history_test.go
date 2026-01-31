package chat

import "testing"

func TestHistoryNavigator_Empty(t *testing.T) {
	h := NewHistoryNavigator()

	// Up on empty history should not crash
	result := h.Up("current")
	if result != "" {
		t.Errorf("Up() on empty history = %q, want empty", result)
	}

	// Down on empty history should not crash
	result = h.Down()
	if result != "" {
		t.Errorf("Down() on empty history = %q, want empty", result)
	}

	// Not browsing
	if h.IsBrowsing() {
		t.Error("IsBrowsing() should be false")
	}
}

func TestHistoryNavigator_Navigation(t *testing.T) {
	h := NewHistoryNavigator()
	h.SetHistory([]string{"first", "second", "third"})

	// Start with no browsing
	if h.IsBrowsing() {
		t.Error("IsBrowsing() should be false initially")
	}

	// Up should start at most recent
	result := h.Up("current")
	if result != "third" {
		t.Errorf("First Up() = %q, want %q", result, "third")
	}
	if !h.IsBrowsing() {
		t.Error("IsBrowsing() should be true after Up")
	}

	// Up again should go older
	result = h.Up("")
	if result != "second" {
		t.Errorf("Second Up() = %q, want %q", result, "second")
	}

	// Up again
	result = h.Up("")
	if result != "first" {
		t.Errorf("Third Up() = %q, want %q", result, "first")
	}

	// Up at oldest should stay at oldest
	result = h.Up("")
	if result != "first" {
		t.Errorf("Fourth Up() = %q, want %q", result, "first")
	}

	// Down should go newer
	result = h.Down()
	if result != "second" {
		t.Errorf("First Down() = %q, want %q", result, "second")
	}

	// Down again
	result = h.Down()
	if result != "third" {
		t.Errorf("Second Down() = %q, want %q", result, "third")
	}

	// Down at newest should restore draft
	result = h.Down()
	if result != "current" {
		t.Errorf("Third Down() = %q, want %q", result, "current")
	}
	if h.IsBrowsing() {
		t.Error("IsBrowsing() should be false after restoring draft")
	}
}

func TestHistoryNavigator_Reset(t *testing.T) {
	h := NewHistoryNavigator()
	h.SetHistory([]string{"first", "second"})

	// Navigate into history
	h.Up("current")
	h.Up("")

	// Reset
	h.Reset()

	if h.IsBrowsing() {
		t.Error("IsBrowsing() should be false after Reset")
	}
	if h.Index() != -1 {
		t.Errorf("Index() after Reset = %d, want -1", h.Index())
	}
}

func TestHistoryNavigator_Add(t *testing.T) {
	h := NewHistoryNavigator()

	// Add entries
	h.Add("first")
	h.Add("second")

	if h.HistoryLen() != 2 {
		t.Errorf("HistoryLen() = %d, want 2", h.HistoryLen())
	}

	// Add duplicate should be skipped
	h.Add("second")
	if h.HistoryLen() != 2 {
		t.Errorf("HistoryLen() after duplicate = %d, want 2", h.HistoryLen())
	}

	// Add different entry should work
	h.Add("third")
	if h.HistoryLen() != 3 {
		t.Errorf("HistoryLen() after new entry = %d, want 3", h.HistoryLen())
	}
}
