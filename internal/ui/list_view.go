package ui

import (
	"fmt"
	"strings"

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

func listColumns(width int) []table.Column {
	titleWidth := width - 50
	if titleWidth < 20 {
		titleWidth = 20
	}
	return []table.Column{
		{Title: " ", Width: 2},
		{Title: "Action", Width: 10},
		{Title: "Pri", Width: 8},
		{Title: "Category", Width: 10},
		{Title: "Info", Width: 14},
		{Title: "Title", Width: titleWidth},
	}
}

func NewListView(width, height int) ListView {
	columns := listColumns(width)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
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

// UpdateTableStyles updates the table styles to match the current theme
func (lv *ListView) UpdateTableStyles(theme Theme) {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(theme.Subtle)).
		BorderBottom(true).
		Bold(true).
		Foreground(lipgloss.Color(theme.Primary))
	s.Selected = s.Selected.
		Foreground(lipgloss.Color(theme.Background)).
		Background(lipgloss.Color(theme.Primary)).
		Bold(false)
	lv.table.SetStyles(s)
}

func (lv *ListView) SetItems(items []Item) {
	lv.items = items
	lv.updateRows()
}

func (lv *ListView) updateRows() {
	rows := make([]table.Row, len(lv.items))
	for i, item := range lv.items {
		sel := " "
		if lv.selected[i] {
			sel = "●"
		}

		actionText := runewidth.FillRight(getActionText(item.Action), 10)
		priorityText := runewidth.FillRight(getPriorityText(item.Priority), 8)
		category := Truncate(item.Category, 10)
		info := formatInfo(item.ReadingTime, item.WordCount)
		title := Truncate(item.Title, lv.width-55)

		rows[i] = table.Row{sel, actionText, priorityText, category, info, title}
	}
	lv.table.SetRows(rows)
}

func formatInfo(readingTime string, wordCount int) string {
	if readingTime != "" && wordCount > 0 {
		return fmt.Sprintf("%s|%dw", Truncate(readingTime, 5), wordCount)
	}
	if readingTime != "" {
		return readingTime
	}
	if wordCount > 0 {
		return fmt.Sprintf("%dw", wordCount)
	}
	return ""
}

func Truncate(s string, maxLen int) string {
	if runewidth.StringWidth(s) > maxLen {
		return runewidth.Truncate(s, maxLen, "…")
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
	case "needs_review":
		return "👁  Review"
	default:
		return "· New"
	}
}

func getPriorityText(priority string) string {
	switch priority {
	case "high":
		return "🔴 High"
	case "medium":
		return "🟡 Med"
	case "low":
		return "🟢 Low"
	default:
		return "  —"
	}
}

// DetailView renders a detail pane for the given item
func (lv *ListView) DetailView(width int, styles Styles) string {
	item := lv.GetItem(lv.cursor)
	if item == nil {
		return ""
	}

	var lines []string

	// Title line
	titleLine := styles.Highlight.Render(Truncate(item.Title, width-4))
	lines = append(lines, titleLine)

	// URL
	if item.URL != "" {
		urlLine := styles.Help.Render(Truncate(item.URL, width-4))
		lines = append(lines, urlLine)
	}

	// Metadata line: source, category, reading time, word count
	var meta []string
	if item.Source != "" {
		meta = append(meta, "src:"+item.Source)
	}
	if item.Category != "" {
		meta = append(meta, "cat:"+item.Category)
	}
	if item.ReadingTime != "" {
		meta = append(meta, item.ReadingTime)
	}
	if item.WordCount > 0 {
		meta = append(meta, fmt.Sprintf("%d words", item.WordCount))
	}
	if len(item.Tags) > 0 {
		meta = append(meta, "tags:"+strings.Join(item.Tags, ","))
	}
	if len(meta) > 0 {
		metaLine := styles.Normal.Render(strings.Join(meta, "  ·  "))
		lines = append(lines, metaLine)
	}

	// Summary (truncated to 2 lines)
	if item.Summary != "" {
		summary := item.Summary
		maxLen := (width - 4) * 2
		if len(summary) > maxLen {
			summary = summary[:maxLen] + "…"
		}
		lines = append(lines, styles.Normal.Render(summary))
	}

	return strings.Join(lines, "\n")
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
	lv.table.SetColumns(listColumns(width))
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
	return "j/k: navigate • x: select • r/l/a/d/n: action • 1/2/3: priority • p: AI triage • enter: edit • q: quit"
}
