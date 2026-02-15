package ui

import (
	"github.com/charmbracelet/huh"
)

// EditForm handles the item editing form using Huh
type EditForm struct {
	form   *huh.Form
	item   *Item
	result *EditResult
}

// EditResult contains the edited values
type EditResult struct {
	Action   string
	Priority string
	Reason   string
	Tags     []string
}

// NewEditForm creates a new edit form for an item
func NewEditForm(item *Item) *EditForm {
	result := &EditResult{
		Action:   item.Action,
		Priority: item.Priority,
		Reason:   "",
		Tags:     []string{},
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Action").
				Options(
					huh.NewOption("Read Now 🔥", "read_now"),
					huh.NewOption("Later ⏰", "later"),
					huh.NewOption("Archive 📁", "archive"),
				).
				Value(&result.Action),

			huh.NewSelect[string]().
				Title("Priority").
				Options(
					huh.NewOption("High 🔴", "high"),
					huh.NewOption("Medium 🟡", "medium"),
					huh.NewOption("Low 🟢", "low"),
					huh.NewOption("None ⚪", ""),
				).
				Value(&result.Priority),
		),
	)

	return &EditForm{
		form:   form,
		item:   item,
		result: result,
	}
}

// Run executes the form and returns the result
func (ef *EditForm) Run() (*EditResult, error) {
	err := ef.form.Run()
	if err != nil {
		return nil, err
	}
	return ef.result, nil
}

// GetForm returns the underlying Huh form for Bubble Tea integration
func (ef *EditForm) GetForm() *huh.Form {
	return ef.form
}

// ApplyResult applies the edit result to the item
func (ef *EditForm) ApplyResult() {
	if ef.item != nil && ef.result != nil {
		ef.item.Action = ef.result.Action
		ef.item.Priority = ef.result.Priority
	}
}
