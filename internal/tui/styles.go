// Package tui provides terminal UI components.
package tui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

// Chat styles
var (
	UserStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	AssistantStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true)
	ErrorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	HelpStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	InputBoxStyle  = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)

	// Autocomplete styles
	AutocompleteBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("63")).
				Padding(0, 1)
	AutocompleteItemStyle     = lipgloss.NewStyle().PaddingLeft(2)
	AutocompleteSelectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))
	AutocompleteDescStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// State-aware styles
	EscWarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B")). // Coral red - urgent but not alarming
			Bold(true)

	EscWarningBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#FF6B6B")). // Match warning color
				Padding(0, 1)

	HistoryModeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#A78BFA")). // Soft purple - "you're in the past"
				Italic(true)

	KeyHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6EE7B7")). // Mint green - actionable
			Bold(true)

	DimHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")) // Slightly brighter than current 241

	HistoryBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#A78BFA")).
				Padding(0, 1)
)

// Picker styles
var (
	TitleStyle        = lipgloss.NewStyle().MarginLeft(2)
	ItemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	SelectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	PaginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	HelpListStyle     = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
)
