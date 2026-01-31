package cmd

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vstratful/openrouter-cli/config"
)

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpListStyle     = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
)

// sessionItem implements list.Item interface
type sessionItem struct {
	summary config.SessionSummary
}

func (i sessionItem) Title() string {
	return i.summary.UpdatedAt.Format("Jan 2, 15:04")
}

func (i sessionItem) Description() string {
	if i.summary.Model != "" {
		return fmt.Sprintf("[%s] \"%s\" (%d messages)", i.summary.Model, i.summary.Preview, i.summary.MessageCount)
	}
	return fmt.Sprintf("\"%s\" (%d messages)", i.summary.Preview, i.summary.MessageCount)
}

func (i sessionItem) FilterValue() string {
	return i.summary.Preview
}

type sessionItemDelegate struct{}

func (d sessionItemDelegate) Height() int                             { return 2 }
func (d sessionItemDelegate) Spacing() int                            { return 1 }
func (d sessionItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d sessionItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(sessionItem)
	if !ok {
		return
	}

	title := i.Title()
	desc := i.Description()

	if index == m.Index() {
		title = selectedItemStyle.Render("> " + title)
		desc = selectedItemStyle.Render("  " + desc)
	} else {
		title = itemStyle.Render(title)
		desc = itemStyle.Render(desc)
	}

	fmt.Fprintf(w, "%s\n%s", title, desc)
}

type sessionPickerModel struct {
	list     list.Model
	selected *config.SessionSummary
	quitting bool
}

func newSessionPickerModel(summaries []config.SessionSummary) sessionPickerModel {
	items := make([]list.Item, len(summaries))
	for i, s := range summaries {
		items[i] = sessionItem{summary: s}
	}

	l := list.New(items, sessionItemDelegate{}, 0, 0)
	l.Title = "Resume a previous session"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpListStyle

	return sessionPickerModel{list: l}
}

func (m sessionPickerModel) Init() tea.Cmd {
	return nil
}

func (m sessionPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 2)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			if i, ok := m.list.SelectedItem().(sessionItem); ok {
				m.selected = &i.summary
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m sessionPickerModel) View() string {
	if m.quitting {
		return ""
	}
	return m.list.View()
}

// runSessionPicker shows the session picker TUI and returns the selected session
func runSessionPicker() (*config.SessionSummary, error) {
	summaries, err := config.ListSessions()
	if err != nil {
		return nil, err
	}

	if len(summaries) == 0 {
		return nil, nil
	}

	model := newSessionPickerModel(summaries)
	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	if m, ok := finalModel.(sessionPickerModel); ok {
		return m.selected, nil
	}

	return nil, nil
}
