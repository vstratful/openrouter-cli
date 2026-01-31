package chat

import "testing"

func TestAutocompleteState_Update(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantVisible bool
		wantCount   int
	}{
		{
			name:        "empty input",
			input:       "",
			wantVisible: false,
			wantCount:   0,
		},
		{
			name:        "non-command input",
			input:       "hello",
			wantVisible: false,
			wantCount:   0,
		},
		{
			name:        "slash only",
			input:       "/",
			wantVisible: true,
			wantCount:   6, // /clear, /exit, /models, /new, /quit, /resume
		},
		{
			name:        "partial command",
			input:       "/e",
			wantVisible: true,
			wantCount:   1, // /exit
		},
		{
			name:        "exact match",
			input:       "/exit",
			wantVisible: false, // exact match hides autocomplete
			wantCount:   1,
		},
		{
			name:        "command with space",
			input:       "/exit arg",
			wantVisible: false,
			wantCount:   0,
		},
		{
			name:        "models prefix",
			input:       "/mo",
			wantVisible: true,
			wantCount:   1, // /models
		},
		{
			name:        "quit prefix",
			input:       "/q",
			wantVisible: true,
			wantCount:   1, // /quit
		},
		{
			name:        "resume prefix",
			input:       "/r",
			wantVisible: true,
			wantCount:   1, // /resume
		},
		{
			name:        "no match",
			input:       "/xyz",
			wantVisible: false,
			wantCount:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAutocompleteState()
			a.Update(tt.input)

			if a.Visible() != tt.wantVisible {
				t.Errorf("Visible() = %v, want %v", a.Visible(), tt.wantVisible)
			}
			if len(a.Filtered()) != tt.wantCount {
				t.Errorf("len(Filtered()) = %d, want %d", len(a.Filtered()), tt.wantCount)
			}
		})
	}
}

func TestAutocompleteState_Navigation(t *testing.T) {
	a := NewAutocompleteState()
	a.Update("/") // Show all 6 commands

	if a.Index() != 0 {
		t.Errorf("Initial Index() = %d, want 0", a.Index())
	}

	// Down navigation
	a.Down()
	if a.Index() != 1 {
		t.Errorf("After Down() Index() = %d, want 1", a.Index())
	}

	a.Down()
	a.Down()
	a.Down()
	a.Down()
	if a.Index() != 5 {
		t.Errorf("After 5x Down() Index() = %d, want 5", a.Index())
	}

	// Down at bottom should stay at bottom
	a.Down()
	if a.Index() != 5 {
		t.Errorf("Down at bottom Index() = %d, want 5", a.Index())
	}

	// Up navigation
	a.Up()
	if a.Index() != 4 {
		t.Errorf("After Up() Index() = %d, want 4", a.Index())
	}

	// Up to top
	a.Up()
	a.Up()
	a.Up()
	a.Up()
	if a.Index() != 0 {
		t.Errorf("After 5x Up() Index() = %d, want 0", a.Index())
	}

	// Up at top should stay at top
	a.Up()
	if a.Index() != 0 {
		t.Errorf("Up at top Index() = %d, want 0", a.Index())
	}
}

func TestAutocompleteState_Select(t *testing.T) {
	a := NewAutocompleteState()
	a.Update("/e")

	selected := a.Select()
	if selected != "/exit" {
		t.Errorf("Select() = %q, want %q", selected, "/exit")
	}
}

func TestAutocompleteState_Hide(t *testing.T) {
	a := NewAutocompleteState()
	a.Update("/")

	if !a.Visible() {
		t.Error("Should be visible after Update")
	}

	a.Hide()

	if a.Visible() {
		t.Error("Should not be visible after Hide")
	}
}

func TestAutocompleteState_IndexClamp(t *testing.T) {
	a := NewAutocompleteState()

	// Start with 6 commands
	a.Update("/")
	a.Down()
	a.Down()
	a.Down()
	a.Down()
	a.Down() // Index = 5

	// Update to show only 1 command
	a.Update("/e") // Only /exit

	// Index should be clamped to 0
	if a.Index() != 0 {
		t.Errorf("Index after narrowing = %d, want 0", a.Index())
	}
}
