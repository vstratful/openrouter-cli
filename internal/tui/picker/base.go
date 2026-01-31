// Package picker provides reusable list picker components.
package picker

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vstratful/openrouter-cli/internal/tui"
)

// Item is the interface for items that can be displayed in a picker.
type Item interface {
	list.Item
	Title() string
	Description() string
}

// ItemDelegate renders items in the picker list.
type ItemDelegate struct{}

func (d ItemDelegate) Height() int                             { return 2 }
func (d ItemDelegate) Spacing() int                            { return 1 }
func (d ItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d ItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(Item)
	if !ok {
		return
	}

	title := i.Title()
	desc := i.Description()

	if index == m.Index() {
		title = tui.SelectedItemStyle.Render("> " + title)
		desc = tui.SelectedItemStyle.Render("  " + desc)
	} else {
		title = tui.ItemStyle.Render(title)
		desc = tui.ItemStyle.Render(desc)
	}

	fmt.Fprintf(w, "%s\n%s", title, desc)
}

// Model is the Bubble Tea model for a generic picker.
type Model struct {
	List     list.Model
	Loading  bool
	Spinner  spinner.Model
	Err      error
	Width    int
	Height   int
	Quitting bool
}

// Config holds configuration for creating a new picker.
type Config struct {
	Title  string
	Items  []list.Item
	Width  int
	Height int
}

// New creates a new picker Model.
func New(cfg Config) Model {
	l := list.New(cfg.Items, ItemDelegate{}, cfg.Width, cfg.Height-2)
	l.Title = cfg.Title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = tui.TitleStyle
	l.Styles.PaginationStyle = tui.PaginationStyle
	l.Styles.HelpStyle = tui.HelpListStyle

	return Model{
		List:   l,
		Width:  cfg.Width,
		Height: cfg.Height,
	}
}

// NewLoading creates a new picker Model in loading state.
func NewLoading(width, height int) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))

	return Model{
		Loading: true,
		Spinner: sp,
		Width:   width,
		Height:  height,
	}
}

// Init initializes the picker.
func (m Model) Init() tea.Cmd {
	if m.Loading {
		return m.Spinner.Tick
	}
	return nil
}

// SetItems sets the items in the picker list.
func (m *Model) SetItems(title string, items []list.Item) {
	l := list.New(items, ItemDelegate{}, m.Width, m.Height-2)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = tui.TitleStyle
	l.Styles.PaginationStyle = tui.PaginationStyle
	l.Styles.HelpStyle = tui.HelpListStyle
	m.List = l
	m.Loading = false
}

// SetError sets an error state.
func (m *Model) SetError(err error) {
	m.Err = err
	m.Loading = false
}

// Update handles messages for the picker.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		if !m.Loading {
			m.List.SetWidth(msg.Width)
			m.List.SetHeight(msg.Height - 2)
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.Quitting = true
			return m, tea.Quit

		case "enter":
			return m, nil // Let caller handle selection
		}

	case spinner.TickMsg:
		if m.Loading {
			var cmd tea.Cmd
			m.Spinner, cmd = m.Spinner.Update(msg)
			return m, cmd
		}
	}

	if !m.Loading && m.Err == nil {
		var cmd tea.Cmd
		m.List, cmd = m.List.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View renders the picker.
func (m Model) View() string {
	if m.Quitting {
		return ""
	}

	if m.Loading {
		return fmt.Sprintf("\n\n   %s Loading...\n", m.Spinner.View())
	}

	if m.Err != nil {
		return tui.ErrorStyle.Render(fmt.Sprintf("\n\n   Error: %s\n", m.Err.Error()))
	}

	return m.List.View()
}

// SelectedItem returns the currently selected item.
func (m Model) SelectedItem() list.Item {
	return m.List.SelectedItem()
}

// IsFiltering returns true if the picker is in filter mode.
func (m Model) IsFiltering() bool {
	return m.List.FilterState() == list.Filtering
}
