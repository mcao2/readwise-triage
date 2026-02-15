package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"
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
		contains string
	}{
		{"Hello World", 5, "Hell"},
		{"Hello", 10, "Hello"},
		{"こんにちは", 5, "こん"},
	}

	for _, tt := range tests {
		got := Truncate(tt.input, tt.max)
		if !strings.Contains(got, tt.contains) {
			t.Errorf("Truncate(%q, %d) = %q, expected to contain %q", tt.input, tt.max, got, tt.contains)
		}
		if runewidth.StringWidth(got) > tt.max {
			t.Errorf("Truncate(%q, %d) = %q, width %d exceeds max", tt.input, tt.max, got, runewidth.StringWidth(got))
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
	if !strings.Contains(getActionText("needs_review"), "Review") {
		t.Error("needs_review should contain 'Review'")
	}
	if !strings.Contains(getActionText(""), "New") {
		t.Error("empty action should contain 'New'")
	}
}

func TestActionTextAlignment(t *testing.T) {
	actions := []string{"read_now", "later", "archive", "delete", "needs_review", ""}

	// All action texts should fit within 10 chars (column width)
	for _, action := range actions {
		text := getActionText(action)
		width := runewidth.StringWidth(text)
		if width > 10 {
			t.Errorf("action '%s' text '%s' has width %d, exceeds column width 10", action, text, width)
		}
	}
}

func TestListView_SetWidthHeight(t *testing.T) {
	lv := NewListView(80, 24)
	lv.SetItems([]Item{{ID: "1", Title: "Test"}})

	lv.SetWidthHeight(120, 40)
	if lv.width != 120 {
		t.Errorf("expected width 120, got %d", lv.width)
	}
	if lv.height != 40 {
		t.Errorf("expected height 40, got %d", lv.height)
	}
}

func TestListView_Init(t *testing.T) {
	lv := NewListView(80, 24)
	cmd := lv.Init()
	if cmd != nil {
		t.Error("expected nil cmd from Init")
	}
}

func TestListView_Update(t *testing.T) {
	lv := NewListView(80, 24)
	lv.SetItems([]Item{{ID: "1", Title: "Test"}, {ID: "2", Title: "Test 2"}})

	updated, cmd := lv.Update(tea.KeyMsg{Type: tea.KeyDown})
	_ = updated
	_ = cmd
	// Just verify it doesn't panic
}

func TestListView_HelpView(t *testing.T) {
	lv := NewListView(80, 24)
	help := lv.helpView()
	if help == "" {
		t.Error("expected non-empty help view")
	}
	if !strings.Contains(help, "navigate") {
		t.Error("expected help to contain 'navigate'")
	}
}

func TestListView_View(t *testing.T) {
	lv := NewListView(80, 24)
	lv.SetItems([]Item{{ID: "1", Title: "Test", Action: "read_now", Priority: "high"}})
	view := lv.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestListView_GetItemOutOfBounds(t *testing.T) {
	lv := NewListView(80, 24)
	lv.SetItems([]Item{{ID: "1", Title: "Test"}})

	if item := lv.GetItem(-1); item != nil {
		t.Error("expected nil for negative index")
	}
	if item := lv.GetItem(5); item != nil {
		t.Error("expected nil for out-of-bounds index")
	}
}

func TestListView_CursorBoundary(t *testing.T) {
	lv := NewListView(80, 24)
	lv.SetItems([]Item{{ID: "1"}, {ID: "2"}})

	// SetCursor out of bounds should be ignored
	lv.SetCursor(10)
	if lv.Cursor() != 0 {
		t.Errorf("expected cursor 0 after out-of-bounds set, got %d", lv.Cursor())
	}

	lv.SetCursor(-1)
	if lv.Cursor() != 0 {
		t.Errorf("expected cursor 0 after negative set, got %d", lv.Cursor())
	}

	// MoveCursor out of bounds should be ignored
	lv.MoveCursor(-1)
	if lv.Cursor() != 0 {
		t.Errorf("expected cursor 0 after negative move, got %d", lv.Cursor())
	}

	lv.MoveCursor(5)
	if lv.Cursor() != 0 {
		t.Errorf("expected cursor 0 after large move, got %d", lv.Cursor())
	}
}

func TestGetPriorityText(t *testing.T) {
	tests := []struct {
		priority string
		contains string
	}{
		{"high", "High"},
		{"medium", "Med"},
		{"low", "Low"},
		{"", "—"},
		{"unknown", "—"},
	}

	for _, tt := range tests {
		text := getPriorityText(tt.priority)
		if !strings.Contains(text, tt.contains) {
			t.Errorf("getPriorityText(%q) = %q, expected to contain %q", tt.priority, text, tt.contains)
		}
	}
}

func TestFormatInfo(t *testing.T) {
	tests := []struct {
		readingTime string
		wordCount   int
		contains    string
		empty       bool
	}{
		{"5 min", 2500, "5 min", false},
		{"5 min", 2500, "2500w", false},
		{"5 min", 0, "5 min", false},
		{"", 2500, "2500w", false},
		{"", 0, "", true},
	}

	for _, tt := range tests {
		got := formatInfo(tt.readingTime, tt.wordCount)
		if tt.empty && got != "" {
			t.Errorf("formatInfo(%q, %d) = %q, expected empty", tt.readingTime, tt.wordCount, got)
		}
		if !tt.empty && !strings.Contains(got, tt.contains) {
			t.Errorf("formatInfo(%q, %d) = %q, expected to contain %q", tt.readingTime, tt.wordCount, got, tt.contains)
		}
	}
}

func TestDetailView(t *testing.T) {
	lv := NewListView(80, 24)
	styles := DefaultStyles()

	items := []Item{
		{
			ID:          "1",
			Title:       "Test Article",
			URL:         "https://example.com/article",
			Summary:     "This is a test summary",
			Category:    "article",
			Source:      "rss",
			WordCount:   1500,
			ReadingTime: "5 min",
			Tags:        []string{"go", "tui"},
		},
	}
	lv.SetItems(items)

	detail := lv.DetailView(80, styles)
	if detail == "" {
		t.Error("expected non-empty detail view")
	}
	if !strings.Contains(detail, "Test Article") {
		t.Error("expected detail to contain title")
	}
	if !strings.Contains(detail, "example.com") {
		t.Error("expected detail to contain URL")
	}
	if !strings.Contains(detail, "test summary") {
		t.Error("expected detail to contain summary")
	}
	if !strings.Contains(detail, "1500 words") {
		t.Error("expected detail to contain word count")
	}
}

func TestDetailViewEmpty(t *testing.T) {
	lv := NewListView(80, 24)
	styles := DefaultStyles()

	// No items — should return empty
	detail := lv.DetailView(80, styles)
	if detail != "" {
		t.Errorf("expected empty detail view, got %q", detail)
	}
}

func TestDetailViewMinimalItem(t *testing.T) {
	lv := NewListView(80, 24)
	styles := DefaultStyles()

	items := []Item{{ID: "1", Title: "Minimal"}}
	lv.SetItems(items)

	detail := lv.DetailView(80, styles)
	if !strings.Contains(detail, "Minimal") {
		t.Error("expected detail to contain title")
	}
}

func TestUpdateTableStyles(t *testing.T) {
	lv := NewListView(80, 24)
	// Should not panic
	lv.UpdateTableStyles(Themes["dracula"])
	lv.UpdateTableStyles(Themes["nord"])
}
