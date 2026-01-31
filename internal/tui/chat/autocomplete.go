package chat

import "strings"

// AutocompleteState manages command autocomplete state.
type AutocompleteState struct {
	// visible indicates whether autocomplete is currently showing
	visible bool

	// index is the currently selected item index
	index int

	// filtered is the list of filtered commands
	filtered []Command
}

// NewAutocompleteState creates a new AutocompleteState.
func NewAutocompleteState() *AutocompleteState {
	return &AutocompleteState{
		visible:  false,
		index:    0,
		filtered: nil,
	}
}

// Update updates the autocomplete state based on the current input.
func (a *AutocompleteState) Update(input string) {
	// Only show autocomplete for input starting with / and no space
	if !strings.HasPrefix(input, "/") || strings.Contains(input, " ") {
		a.visible = false
		a.filtered = nil
		a.index = 0
		return
	}

	a.filtered = FilterCommands(input)

	// Don't show autocomplete if input exactly matches a command
	exactMatch := false
	for _, cmd := range a.filtered {
		if strings.EqualFold(cmd.Name, input) {
			exactMatch = true
			break
		}
	}

	a.visible = len(a.filtered) > 0 && !exactMatch

	// Clamp index to valid range
	if a.index >= len(a.filtered) {
		a.index = max(0, len(a.filtered)-1)
	}
}

// Visible returns whether autocomplete is currently showing.
func (a *AutocompleteState) Visible() bool {
	return a.visible
}

// Hide hides the autocomplete dropdown.
func (a *AutocompleteState) Hide() {
	a.visible = false
}

// Up moves selection up in the autocomplete list.
func (a *AutocompleteState) Up() {
	if a.index > 0 {
		a.index--
	}
}

// Down moves selection down in the autocomplete list.
func (a *AutocompleteState) Down() {
	if a.index < len(a.filtered)-1 {
		a.index++
	}
}

// Select returns the currently selected command name.
// Returns empty string if no valid selection.
func (a *AutocompleteState) Select() string {
	if a.index < len(a.filtered) {
		return a.filtered[a.index].Name
	}
	return ""
}

// Index returns the current selection index.
func (a *AutocompleteState) Index() int {
	return a.index
}

// Filtered returns the filtered commands.
func (a *AutocompleteState) Filtered() []Command {
	return a.filtered
}
