package picker

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/list"
	"github.com/vstratful/openrouter-cli/internal/api"
)

// FormatPricePerMillion converts a price-per-token string to a formatted price per million tokens.
func FormatPricePerMillion(pricePerToken string) string {
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

// ModelItem wraps a Model for display in a picker.
type ModelItem struct {
	Model api.Model
}

func (i ModelItem) Title() string {
	return i.Model.ID
}

func (i ModelItem) Description() string {
	var desc string
	if i.Model.Name != "" && i.Model.Name != i.Model.ID {
		desc = i.Model.Name
	}

	if i.Model.ContextLength != nil {
		if desc != "" {
			desc += " | "
		}
		desc += fmt.Sprintf("%dk ctx", *i.Model.ContextLength/1000)
	}

	if i.Model.Pricing.Prompt != "" || i.Model.Pricing.Completion != "" {
		if desc != "" {
			desc += " | "
		}
		desc += fmt.Sprintf("$%s/$%s per 1M tokens", FormatPricePerMillion(i.Model.Pricing.Prompt), FormatPricePerMillion(i.Model.Pricing.Completion))
	}

	return desc
}

func (i ModelItem) FilterValue() string {
	return i.Model.ID + " " + i.Model.Name
}

// NewModelPicker creates a new picker for models in loading state.
func NewModelPicker(width, height int) Model {
	return NewLoading(width, height)
}

// SetModels sets the models in the picker.
func SetModels(m *Model, models []api.Model) {
	items := make([]list.Item, len(models))
	for i, model := range models {
		items[i] = ModelItem{Model: model}
	}
	m.SetItems("Select a model", items)
}

// GetModel extracts the Model from a selected item.
func GetModel(item list.Item) *api.Model {
	if mi, ok := item.(ModelItem); ok {
		return &mi.Model
	}
	return nil
}

// HasTextModality checks if "text" is in the list of modalities.
func HasTextModality(modalities []string) bool {
	for _, m := range modalities {
		if m == "text" {
			return true
		}
	}
	return false
}

// FilterTextModels filters models to only those with text input and output.
func FilterTextModels(models []api.Model) []api.Model {
	filtered := make([]api.Model, 0, len(models))
	for _, m := range models {
		if HasTextModality(m.Architecture.InputModalities) && HasTextModality(m.Architecture.OutputModalities) {
			filtered = append(filtered, m)
		}
	}
	return filtered
}
