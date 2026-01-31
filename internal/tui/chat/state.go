package chat

import "time"

// ChatState represents the current state of the chat UI.
type ChatState int

const (
	// StateIdle is the default state, ready for user input.
	StateIdle ChatState = iota
	// StateStreaming indicates an active streaming response.
	StateStreaming
	// StateEscPending indicates waiting for a second ESC press.
	StateEscPending
)

// EscAction represents the action to take on double ESC press.
type EscAction int

const (
	// EscActionClear clears the input.
	EscActionClear EscAction = iota
	// EscActionExit exits the application.
	EscActionExit
)

// escState holds state for ESC double-press handling.
type escState struct {
	pressedAt time.Time
	action    EscAction
}

// String returns a human-readable name for the state.
func (s ChatState) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateStreaming:
		return "streaming"
	case StateEscPending:
		return "esc_pending"
	default:
		return "unknown"
	}
}
