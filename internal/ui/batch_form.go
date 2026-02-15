package ui

import (
	"github.com/charmbracelet/huh"
)

type BatchForm struct {
	form   *huh.Form
	result *BatchResult
}

type BatchResult struct {
	FilterAction string
	NewAction    string
	NewPriority  string
	AddTags      []string
	RemoveTags   []string
}

func NewBatchForm() *BatchForm {
	result := &BatchResult{}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Filter by current action (optional)").
				Options(
					huh.NewOption("All items", ""),
					huh.NewOption("Read Now", "read_now"),
					huh.NewOption("Later", "later"),
					huh.NewOption("Archive", "archive"),
					huh.NewOption("Delete", "delete"),
				).
				Value(&result.FilterAction),

			huh.NewSelect[string]().
				Title("Set action to").
				Options(
					huh.NewOption("Keep current", ""),
					huh.NewOption("Read Now 🔥", "read_now"),
					huh.NewOption("Later ⏰", "later"),
					huh.NewOption("Archive 📁", "archive"),
					huh.NewOption("Delete 🗑️", "delete"),
				).
				Value(&result.NewAction),

			huh.NewSelect[string]().
				Title("Set priority to").
				Options(
					huh.NewOption("Keep current", ""),
					huh.NewOption("High 🔴", "high"),
					huh.NewOption("Medium 🟡", "medium"),
					huh.NewOption("Low 🟢", "low"),
				).
				Value(&result.NewPriority),
		),
	)

	return &BatchForm{
		form:   form,
		result: result,
	}
}

func (bf *BatchForm) Run() (*BatchResult, error) {
	err := bf.form.Run()
	if err != nil {
		return nil, err
	}
	return bf.result, nil
}

func (bf *BatchForm) GetForm() *huh.Form {
	return bf.form
}

func (bf *BatchForm) ApplyToItems(items []Item, selectedIndices []int) int {
	if bf.result == nil {
		return 0
	}

	changedCount := 0
	for _, idx := range selectedIndices {
		if idx >= len(items) {
			continue
		}

		item := &items[idx]

		if bf.result.FilterAction != "" && item.Action != bf.result.FilterAction {
			continue
		}

		if bf.result.NewAction != "" {
			item.Action = bf.result.NewAction
			changedCount++
		}
		if bf.result.NewPriority != "" {
			item.Priority = bf.result.NewPriority
			changedCount++
		}
	}

	return changedCount
}
