package cmd

import (
	"fmt"
	"io"
	"strconv"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// formatPricePerMillion converts a price-per-token string to a formatted price per million tokens
func formatPricePerMillion(pricePerToken string) string {
	price, err := strconv.ParseFloat(pricePerToken, 64)
	if err != nil || price == 0 {
		if pricePerToken == "0" {
			return "0"
		}
		return pricePerToken
	}
	pricePerMillion := price * 1_000_000
	if pricePerMillion < 0.01 {
		return fmt.Sprintf("%.4f", pricePerMillion)
	}
	return fmt.Sprintf("%.2f", pricePerMillion)
}

// modelItem implements list.Item interface for the model picker
type modelItem struct {
	model Model
}

func (i modelItem) Title() string {
	return i.model.ID
}

func (i modelItem) Description() string {
	var desc string
	if i.model.Name != "" && i.model.Name != i.model.ID {
		desc = i.model.Name
	}

	if i.model.ContextLength != nil {
		if desc != "" {
			desc += " | "
		}
		desc += fmt.Sprintf("%dk ctx", *i.model.ContextLength/1000)
	}

	if i.model.Pricing.Prompt != "" || i.model.Pricing.Completion != "" {
		if desc != "" {
			desc += " | "
		}
		desc += fmt.Sprintf("$%s/$%s per 1M tokens", formatPricePerMillion(i.model.Pricing.Prompt), formatPricePerMillion(i.model.Pricing.Completion))
	}

	return desc
}

func (i modelItem) FilterValue() string {
	return i.model.ID + " " + i.model.Name
}

// modelItemDelegate handles rendering of model items in the list
type modelItemDelegate struct{}

func (d modelItemDelegate) Height() int                             { return 2 }
func (d modelItemDelegate) Spacing() int                            { return 1 }
func (d modelItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d modelItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(modelItem)
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

// modelPickerModel is the Bubble Tea model for the model picker
type modelPickerModel struct {
	list     list.Model
	selected *Model
	loading  bool
	spinner  spinner.Model
	err      error
	width    int
	height   int
}

// Message types for async model loading
type modelsLoadedMsg struct {
	models []Model
}

type modelsLoadErrorMsg struct {
	err error
}

// loadModelsCmd fetches models asynchronously from the API
func loadModelsCmd(apiKey string) tea.Cmd {
	return func() tea.Msg {
		models, err := GetModels(apiKey, nil)
		if err != nil {
			return modelsLoadErrorMsg{err: err}
		}

		// Filter to only models with text input and output modalities
		filtered := make([]Model, 0, len(models))
		for _, m := range models {
			if hasTextModality(m.Architecture.InputModalities) && hasTextModality(m.Architecture.OutputModalities) {
				filtered = append(filtered, m)
			}
		}

		return modelsLoadedMsg{models: filtered}
	}
}

// hasTextModality checks if "text" is in the list of modalities
func hasTextModality(modalities []string) bool {
	for _, m := range modalities {
		if m == "text" {
			return true
		}
	}
	return false
}

// newModelPickerModel creates a new model picker in loading state
func newModelPickerModel(width, height int) modelPickerModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))

	return modelPickerModel{
		loading: true,
		spinner: sp,
		width:   width,
		height:  height,
	}
}

func (m modelPickerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m modelPickerModel) Update(msg tea.Msg) (modelPickerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case modelsLoadedMsg:
		items := make([]list.Item, len(msg.models))
		for i, model := range msg.models {
			items[i] = modelItem{model: model}
		}

		l := list.New(items, modelItemDelegate{}, m.width, m.height-2)
		l.Title = "Select a model"
		l.SetShowStatusBar(false)
		l.SetFilteringEnabled(true)
		l.Styles.Title = titleStyle
		l.Styles.PaginationStyle = paginationStyle
		l.Styles.HelpStyle = helpListStyle

		m.list = l
		m.loading = false
		return m, nil

	case modelsLoadErrorMsg:
		m.err = msg.err
		m.loading = false
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	if !m.loading && m.err == nil {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m modelPickerModel) View() string {
	if m.loading {
		return fmt.Sprintf("\n\n   %s Loading models...\n", m.spinner.View())
	}

	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("\n\n   Error loading models: %s\n", m.err.Error()))
	}

	return m.list.View()
}
