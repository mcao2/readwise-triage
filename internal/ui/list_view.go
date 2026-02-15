package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

type ListView struct {
	table    table.Model
	items    []Item
	cursor   int
	selected map[int]bool
	width    int
	height   int
}

func NewListView(width, height int) ListView {
	columns := []table.Column{
		{Title: " ", Width: 3},
		{Title: "Action", Width: 12},
		{Title: "Priority", Width: 12},
		{Title: "Category", Width: 12},
		{Title: "Info", Width: 18},
		{Title: "Title", Width: width - 65},
	}

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)

	t := table.New(
		table.WithColumns(columns),
		table.WithHeight(height-4),
		table.WithFocused(true),
	)
	t.SetStyles(s)

	return ListView{
		table:    t,
		selected: make(map[int]bool),
		width:    width,
		height:   height,
	}
}

func (lv *ListView) SetItems(items []Item) {
	lv.items = items
	lv.updateRows()
}

func (lv *ListView) updateRows() {
	rows := make([]table.Row, len(lv.items))
	for i, item := range lv.items {
		selected := "[ ]"
		if lv.selected[i] {
			selected = "[x]"
		}

		actionText := runewidth.FillRight(getActionText(item.Action), 12)
		priorityText := runewidth.FillRight(getPriorityText(item.Priority), 12)
		category := Truncate(item.Category, 12)
		info := fmt.Sprintf("%s | %dw", Truncate(item.ReadingTime, 6), item.WordCount)
		title := Truncate(item.Title, lv.width-70)

		rows[i] = table.Row{selected, actionText, priorityText, category, info, title}
	}
	lv.table.SetRows(rows)
}

func Truncate(s string, maxLen int) string {
	if runewidth.StringWidth(s) > maxLen {
		return runewidth.Truncate(s, maxLen, "...")
	}
	return s
}

func getActionText(action string) string {
	switch action {
	case "read_now":
		return "🔥 Read"
	case "later":
		return "⏰ Later"
	case "archive":
		return "📁 Archive"
	case "delete":
		return "❌ Delete"
	default:
		return "❓ New"
	}
}

func getPriorityText(priority string) string {
	switch priority {
	case "high":
		return "🔴 High"
	case "medium":
		return "🟡 Medium"
	case "low":
		return "🟢 Low"
	default:
		return "⚪ None"
	}
}

func (lv ListView) Cursor() int {
	return lv.cursor
}

func (lv *ListView) SetCursor(pos int) {
	if pos >= 0 && pos < len(lv.items) {
		lv.cursor = pos
		lv.table.SetCursor(pos)
	}
}

func (lv *ListView) MoveCursor(delta int) {
	newPos := lv.cursor + delta
	if newPos >= 0 && newPos < len(lv.items) {
		lv.cursor = newPos
		lv.table.SetCursor(newPos)
	}
}

func (lv *ListView) ToggleSelection() {
	if lv.cursor < len(lv.items) {
		lv.selected[lv.cursor] = !lv.selected[lv.cursor]
		lv.updateRows()
	}
}

func (lv ListView) IsSelected(index int) bool {
	return lv.selected[index]
}

func (lv ListView) GetSelected() []int {
	var indices []int
	for i, selected := range lv.selected {
		if selected {
			indices = append(indices, i)
		}
	}
	return indices
}

func (lv ListView) GetItem(index int) *Item {
	if index >= 0 && index < len(lv.items) {
		return &lv.items[index]
	}
	return nil
}

func (lv ListView) View() string {
	return lv.table.View()
}

func (lv *ListView) SetWidthHeight(width, height int) {
	lv.width = width
	lv.height = height
	lv.table.SetHeight(height - 4)

	columns := []table.Column{
		{Title: " ", Width: 3},
		{Title: "Action", Width: 12},
		{Title: "Priority", Width: 12},
		{Title: "Category", Width: 12},
		{Title: "Info", Width: 18},
		{Title: "Title", Width: width - 65},
	}
	lv.table.SetColumns(columns)
}

func (lv ListView) Init() tea.Cmd {
	return nil
}

func (lv ListView) Update(msg tea.Msg) (ListView, tea.Cmd) {
	var cmd tea.Cmd
	lv.table, cmd = lv.table.Update(msg)
	return lv, cmd
}

func (lv ListView) helpView() string {
	return "j/k: navigate • x: select • r/l/a/d: action • 1/2/3: priority • p: AI triage • enter: edit • q: quit"
}
