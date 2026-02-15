package ui

import (
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

// ListView handles the item list display and navigation
type ListView struct {
	table    table.Model
	items    []Item
	cursor   int
	selected map[int]bool
	width    int
	height   int
}

// NewListView creates a new list view
func NewListView(width, height int) ListView {
	columns := []table.Column{
		{Title: "", Width: 2}, // Selection indicator
		{Title: "", Width: 2}, // Action icon
		{Title: "", Width: 2}, // Priority icon
		{Title: "Title", Width: width - 20},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithHeight(height-4),
	)

	return ListView{
		table:    t,
		selected: make(map[int]bool),
		width:    width,
		height:   height,
	}
}

// SetItems updates the items in the list
func (lv *ListView) SetItems(items []Item) {
	lv.items = items
	lv.updateRows()
}

// updateRows refreshes the table rows
func (lv *ListView) updateRows() {
	rows := make([]table.Row, len(lv.items))
	for i, item := range lv.items {
		selected := " "
		if lv.selected[i] {
			selected = "x"
		}

		actionIcon := getActionIcon(item.Action)
		priorityIcon := getPriorityIcon(item.Priority)

		// Truncate title if too long
		title := item.Title
		maxLen := lv.width - 25
		if len(title) > maxLen {
			title = title[:maxLen-3] + "..."
		}

		rows[i] = table.Row{selected, actionIcon, priorityIcon, title}
	}
	lv.table.SetRows(rows)
}

// Cursor returns the current cursor position
func (lv ListView) Cursor() int {
	return lv.cursor
}

// SetCursor sets the cursor position
func (lv *ListView) SetCursor(pos int) {
	if pos >= 0 && pos < len(lv.items) {
		lv.cursor = pos
		lv.table.SetCursor(pos)
	}
}

// MoveCursor moves the cursor by delta
func (lv *ListView) MoveCursor(delta int) {
	newPos := lv.cursor + delta
	if newPos >= 0 && newPos < len(lv.items) {
		lv.cursor = newPos
		lv.table.SetCursor(newPos)
	}
}

// ToggleSelection toggles selection at cursor
func (lv *ListView) ToggleSelection() {
	if lv.cursor < len(lv.items) {
		lv.selected[lv.cursor] = !lv.selected[lv.cursor]
		lv.updateRows()
	}
}

// IsSelected checks if an item is selected
func (lv ListView) IsSelected(index int) bool {
	return lv.selected[index]
}

// GetSelected returns all selected indices
func (lv ListView) GetSelected() []int {
	var indices []int
	for i, selected := range lv.selected {
		if selected {
			indices = append(indices, i)
		}
	}
	return indices
}

// GetItem returns the item at index
func (lv ListView) GetItem(index int) *Item {
	if index >= 0 && index < len(lv.items) {
		return &lv.items[index]
	}
	return nil
}

// View renders the list view
func (lv ListView) View() string {
	return lv.table.View()
}

// SetWidthHeight updates dimensions
func (lv *ListView) SetWidthHeight(width, height int) {
	lv.width = width
	lv.height = height
	lv.table.SetHeight(height - 4)

	// Update column widths
	columns := []table.Column{
		{Title: "", Width: 2},
		{Title: "", Width: 2},
		{Title: "", Width: 2},
		{Title: "Title", Width: width - 20},
	}
	lv.table.SetColumns(columns)
}

// Init initializes the list view
func (lv ListView) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (lv ListView) Update(msg tea.Msg) (ListView, tea.Cmd) {
	var cmd tea.Cmd
	lv.table, cmd = lv.table.Update(msg)
	return lv, cmd
}

func getActionIcon(action string) string {
	switch action {
	case "read_now":
		return "🔥"
	case "later":
		return "⏰"
	case "archive":
		return "📁"
	case "delete":
		return "🗑️"
	default:
		return "❓"
	}
}

func getPriorityIcon(priority string) string {
	switch priority {
	case "high":
		return "🔴"
	case "medium":
		return "🟡"
	case "low":
		return "🟢"
	default:
		return "⚪"
	}
}

func (lv ListView) helpView() string {
	return "j/k: navigate • x: select • r/l/a/d: action • 1/2/3: priority • enter: edit • q: quit"
}
