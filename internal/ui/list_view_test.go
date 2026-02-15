package ui

import (
	"strings"
	"testing"
)

func TestListView_SetItems(t *testing.T) {
	lv := NewListView(80, 20)
	items := []Item{
		{ID: "1", Title: "Item 1", Action: "read_now", Priority: "high"},
		{ID: "2", Title: "Item 2", Action: "later", Priority: "medium"},
	}
	lv.SetItems(items)

	if len(lv.items) != 2 {
		t.Errorf("expected 2 items, got %d", len(lv.items))
	}

	item1 := lv.GetItem(0)
	if item1.Title != "Item 1" {
		t.Errorf("expected Item 1, got %s", item1.Title)
	}
}

func TestListView_Selection(t *testing.T) {
	lv := NewListView(80, 20)
	lv.SetItems([]Item{{ID: "1"}, {ID: "2"}})

	lv.SetCursor(0)
	lv.ToggleSelection()
	if !lv.IsSelected(0) {
		t.Error("expected item 0 to be selected")
	}

	lv.MoveCursor(1)
	lv.ToggleSelection()
	if !lv.IsSelected(1) {
		t.Error("expected item 1 to be selected")
	}

	selected := lv.GetSelected()
	if len(selected) != 2 {
		t.Errorf("expected 2 selected items, got %d", len(selected))
	}

	lv.ToggleSelection()
	if lv.IsSelected(1) {
		t.Error("expected item 1 to be deselected")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		max      int
		expected string
	}{
		{"Hello World", 5, "He..."},
		{"Hello", 10, "Hello"},
		{"こんにちは", 5, "こ..."},
	}

	for _, tt := range tests {
		got := Truncate(tt.input, tt.max)
		if got != tt.expected {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.expected)
		}
	}
}

func TestGetActionText(t *testing.T) {
	if !strings.Contains(getActionText("read_now"), "Read") {
		t.Error("read_now should contain 'Read'")
	}
	if !strings.Contains(getActionText("later"), "Later") {
		t.Error("later should contain 'Later'")
	}
	if !strings.Contains(getActionText("archive"), "Archive") {
		t.Error("archive should contain 'Archive'")
	}
	if !strings.Contains(getActionText("delete"), "Delete") {
		t.Error("delete should contain 'Delete'")
	}
	if !strings.Contains(getActionText(""), "New") {
		t.Error("empty action should contain 'New'")
	}
}
