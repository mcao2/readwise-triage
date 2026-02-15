package ui

import "testing"

func TestBatchForm_ApplyToItems(t *testing.T) {
	bf := &BatchForm{
		result: &BatchResult{
			NewAction:   "read_now",
			NewPriority: "high",
		},
	}

	items := []Item{
		{ID: "1", Action: "later", Priority: "low"},
		{ID: "2", Action: "later", Priority: "low"},
		{ID: "3", Action: "later", Priority: "low"},
	}

	selected := []int{0, 2}
	changed := bf.ApplyToItems(items, selected)

	if changed != 4 {
		t.Errorf("expected 4 changes, got %d", changed)
	}

	if items[0].Action != "read_now" || items[0].Priority != "high" {
		t.Errorf("item 0 not updated correctly: %+v", items[0])
	}
	if items[1].Action != "later" {
		t.Errorf("item 1 should not have changed: %+v", items[1])
	}
	if items[2].Action != "read_now" || items[2].Priority != "high" {
		t.Errorf("item 2 not updated correctly: %+v", items[2])
	}
}

func TestBatchForm_ApplyWithFilter(t *testing.T) {
	bf := &BatchForm{
		result: &BatchResult{
			FilterAction: "read_now",
			NewAction:    "archive",
		},
	}

	items := []Item{
		{ID: "1", Action: "read_now"},
		{ID: "2", Action: "later"},
	}

	selected := []int{0, 1}
	changed := bf.ApplyToItems(items, selected)

	if changed != 1 {
		t.Errorf("expected 1 change, got %d", changed)
	}

	if items[0].Action != "archive" {
		t.Errorf("item 0 should have changed to archive, got %s", items[0].Action)
	}
	if items[1].Action != "later" {
		t.Errorf("item 1 should still be later, got %s", items[1].Action)
	}
}
