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
	table      table.Model
	items      []Item
	cursor     int
	selected   map[int]bool
	width      int
	height     int
	visibleRows int // number of data rows visible (excluding header)

	// Styles for custom rendering
	headerStyle   lipgloss.Style
	cellStyle     lipgloss.Style
	selectedStyle lipgloss.Style
	columns       []table.Column
}

func listColumns(width int) []table.Column {
	// Each cell has Padding(0,1) adding 2 chars per column (7 columns = 14 extra).
	// Subtract 2 more to avoid hitting exact terminal width (causes implicit wraps).
	fixedWidth := 2 + 10 + 8 + 10 + 14 + 20 // non-title columns
	padding := 7*2 + 2                        // 7 columns × 2 chars padding each + 2 safety margin
	titleWidth := width - fixedWidth - padding
	if titleWidth < 20 {
		titleWidth = 20
	}
	return []table.Column{
		{Title: " ", Width: 2},
		{Title: "Action", Width: 10},
		{Title: "Priority", Width: 8},
		{Title: "Category", Width: 10},
		{Title: "Info", Width: 14},
		{Title: "Tags", Width: 20},
		{Title: "Title", Width: titleWidth},
	}
}

func NewListView(width, height int) ListView {
	columns := listColumns(width)

	headerStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	cellStyle := lipgloss.NewStyle().Padding(0, 1)

	// Reserve space for: header(2) + divider(1) + detail pane(4) + status(1) + footer(4)
	visibleRows := height - 12
	// Subtract 2 for the table header (text + border)
	visibleRows -= 2
	if visibleRows < 3 {
		visibleRows = 3
	}

	// Still create the table for compatibility but we won't use its View()
	t := table.New(
		table.WithColumns(columns),
		table.WithHeight(visibleRows+2),
		table.WithFocused(true),
	)

	return ListView{
		table:         t,
		selected:      make(map[int]bool),
		width:         width,
		height:        height,
		visibleRows:   visibleRows,
		headerStyle:   headerStyle,
		cellStyle:     cellStyle,
		selectedStyle: selectedStyle,
		columns:       columns,
	}
}

// UpdateTableStyles updates the styles to match the current theme
func (lv *ListView) UpdateTableStyles(theme Theme) {
	lv.headerStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(theme.Subtle)).
		BorderBottom(true).
		Bold(true).
		Foreground(lipgloss.Color(theme.Primary))
	lv.selectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Background)).
		Background(lipgloss.Color(theme.Primary)).
		Bold(false)

	// Keep the bubbles table in sync for any code that still uses it
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
		tags := Truncate(strings.Join(item.Tags, ", "), 20)
		title := Truncate(item.Title, lv.width-80)

		rows[i] = table.Row{sel, actionText, priorityText, category, info, tags, title}
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

// detailPaneHeight is the fixed number of lines the detail pane always occupies.
const detailPaneHeight = 4

// DetailView renders a detail pane for the given item, padded to a fixed height.
func (lv *ListView) DetailView(width int, styles Styles) string {
	item := lv.GetItem(lv.cursor)
	if item == nil {
		return ""
	}

	maxWidth := width - 4
	if maxWidth < 20 {
		maxWidth = 20
	}

	var lines []string

	lines = append(lines, styles.Highlight.Render(Truncate(item.Title, maxWidth)))

	if item.URL != "" {
		lines = append(lines, styles.Help.Render(Truncate(item.URL, maxWidth)))
	}

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
		lines = append(lines, styles.Normal.Render(Truncate(strings.Join(meta, " · "), maxWidth)))
	}

	if item.Summary != "" {
		lines = append(lines, styles.HelpDesc.Render(Truncate(item.Summary, maxWidth)))
	}

	for len(lines) < detailPaneHeight {
		lines = append(lines, "")
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

// UpdateTable forwards a message to the underlying table component
func (lv *ListView) UpdateTable(msg tea.Msg) {
	lv.table, _ = lv.table.Update(msg)
}

// SyncCursor reads the table's internal cursor and syncs our cursor to it
func (lv *ListView) SyncCursor() int {
	lv.cursor = lv.table.Cursor()
	return lv.cursor
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

// renderCell renders a single cell value with the given column width.
func (lv *ListView) renderCell(value string, colWidth int) string {
	style := lipgloss.NewStyle().Width(colWidth).MaxWidth(colWidth).Inline(true)
	return lv.cellStyle.Render(style.Render(runewidth.Truncate(value, colWidth, "…")))
}

// View renders the table with our own scrolling logic, bypassing the
// bubbles table viewport which has broken YOffset calculations.
func (lv ListView) View() string {
	rows := lv.table.Rows()

	// Render header
	headerCells := make([]string, 0, len(lv.columns))
	for _, col := range lv.columns {
		if col.Width <= 0 {
			continue
		}
		style := lipgloss.NewStyle().Width(col.Width).MaxWidth(col.Width).Inline(true)
		cell := style.Render(runewidth.Truncate(col.Title, col.Width, "…"))
		headerCells = append(headerCells, lv.headerStyle.Render(lv.cellStyle.Render(cell)))
	}
	header := lipgloss.JoinHorizontal(lipgloss.Top, headerCells...)

	// Calculate visible window
	visibleRows := lv.visibleRows
	if visibleRows <= 0 {
		visibleRows = 10
	}

	start := 0
	if lv.cursor >= visibleRows {
		start = lv.cursor - visibleRows + 1
	}
	end := start + visibleRows
	if end > len(rows) {
		end = len(rows)
		start = end - visibleRows
		if start < 0 {
			start = 0
		}
	}

	// Render visible rows
	renderedRows := make([]string, 0, visibleRows)
	for i := start; i < end; i++ {
		cells := make([]string, 0, len(lv.columns))
		for ci, value := range rows[i] {
			if lv.columns[ci].Width <= 0 {
				continue
			}
			cells = append(cells, lv.renderCell(value, lv.columns[ci].Width))
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, cells...)
		if i == lv.cursor {
			row = lv.selectedStyle.Render(row)
		}
		renderedRows = append(renderedRows, row)
	}

	// Pad to fixed height
	for len(renderedRows) < visibleRows {
		renderedRows = append(renderedRows, "")
	}

	return header + "\n" + strings.Join(renderedRows, "\n")
}

func (lv *ListView) SetWidthHeight(width, height int) {
	lv.width = width
	lv.height = height
	lv.columns = listColumns(width)

	// Reserve space for: header(2) + divider(1) + detail pane(4) + status(1) + footer(4)
	visibleRows := height - 12
	// Subtract 2 for the table header (text + border)
	visibleRows -= 2
	if visibleRows < 3 {
		visibleRows = 3
	}
	lv.visibleRows = visibleRows

	lv.table.SetHeight(visibleRows + 2)
	lv.table.SetColumns(lv.columns)
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
