package chat

// HistoryNavigator manages navigation through input history.
type HistoryNavigator struct {
	// history is the list of previous inputs
	history []string

	// index is the current position in history (-1 = not browsing)
	index int

	// draft is the current input being typed before navigating history
	draft string
}

// NewHistoryNavigator creates a new HistoryNavigator.
func NewHistoryNavigator() *HistoryNavigator {
	return &HistoryNavigator{
		history: []string{},
		index:   -1,
		draft:   "",
	}
}

// SetHistory sets the history list.
func (h *HistoryNavigator) SetHistory(history []string) {
	h.history = history
}

// IsBrowsing returns true if currently browsing history.
func (h *HistoryNavigator) IsBrowsing() bool {
	return h.index >= 0
}

// Index returns the current history index.
func (h *HistoryNavigator) Index() int {
	return h.index
}

// HistoryLen returns the length of history.
func (h *HistoryNavigator) HistoryLen() int {
	return len(h.history)
}

// Up navigates to an older history entry.
// Returns the history entry to display, or empty string if no change.
func (h *HistoryNavigator) Up(currentInput string) string {
	if len(h.history) == 0 {
		return ""
	}

	// First press: save current draft and start at most recent
	if h.index == -1 {
		h.draft = currentInput
		h.index = len(h.history) - 1
	} else if h.index > 0 {
		// Move to older entry
		h.index--
	}

	return h.history[h.index]
}

// Down navigates to a newer history entry.
// Returns the history entry to display, or the draft if at bottom.
func (h *HistoryNavigator) Down() string {
	if h.index == -1 {
		return ""
	}

	if h.index < len(h.history)-1 {
		// Move to newer entry
		h.index++
		return h.history[h.index]
	}

	// At bottom of history, restore draft
	h.index = -1
	return h.draft
}

// Reset resets the history navigation state.
func (h *HistoryNavigator) Reset() {
	h.index = -1
	h.draft = ""
}

// Add adds an entry to history (skip consecutive duplicates).
func (h *HistoryNavigator) Add(entry string) {
	if len(h.history) == 0 || h.history[len(h.history)-1] != entry {
		h.history = append(h.history, entry)
	}
}
