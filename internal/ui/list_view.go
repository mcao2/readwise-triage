package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// Column defines a table column for custom rendering.
type Column struct {
	Title string
	Width int
}

type ListView struct {
	items       []Item
	rows        [][]string
	cursor      int
	selected    map[int]bool
	width       int
	height      int
	visibleRows int
	columns     []Column

	// Styles for custom rendering
	headerStyle   lipgloss.Style
	cellStyle     lipgloss.Style
	selectedStyle lipgloss.Style
}

func listColumns(width int) []Column {
	fixedWidth := 2 + 10 + 10 + 14 + 20 // non-title columns
	padding := 6*2 + 2                  // 6 columns × 2 chars padding each + 2 safety margin
	titleWidth := width - fixedWidth - padding
	if titleWidth < 20 {
		titleWidth = 20
	}
	return []Column{
		{Title: " ", Width: 2},
		{Title: "Action", Width: 10},
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

	visibleRows := height - 12 - 2 // Reserve 14 lines: header(2) + detail(4) + status(1) + footer(4) + table_header(2) + gap(1)
	if visibleRows < 3 {
		visibleRows = 3
	}

	return ListView{
		selected:      make(map[int]bool),
		width:         width,
		height:        height,
		visibleRows:   visibleRows,
		columns:       columns,
		headerStyle:   headerStyle,
		cellStyle:     cellStyle,
		selectedStyle: selectedStyle,
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
}

func (lv *ListView) SetItems(items []Item) {
	lv.items = items
	lv.updateRows()
}

func (lv *ListView) updateRows() {
	rows := make([][]string, len(lv.items))
	for i, item := range lv.items {
		sel := " "
		if lv.selected[i] {
			sel = "●"
		}
		actionText := runewidth.FillRight(getActionText(item.Action), 10)
		category := Truncate(item.Category, 10)
		info := formatInfo(item.ReadingTime, item.WordCount)
		tags := Truncate(strings.Join(item.Tags, ", "), 20)
		title := Truncate(item.Title, lv.width-70)
		rows[i] = []string{sel, actionText, category, info, tags, title}
	}
	lv.rows = rows
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
	default:
		return "· New"
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

func (lv *ListView) Cursor() int {
	return lv.cursor
}

func (lv *ListView) SetCursor(pos int) {
	if pos >= 0 && pos < len(lv.items) {
		lv.cursor = pos
	}
}

func (lv *ListView) MoveCursor(delta int) {
	newPos := lv.cursor + delta
	if newPos >= 0 && newPos < len(lv.items) {
		lv.cursor = newPos
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

// renderCell renders a single cell value with the given column width.
func (lv *ListView) renderCell(value string, colWidth int) string {
	style := lipgloss.NewStyle().Width(colWidth).MaxWidth(colWidth).Inline(true)
	return lv.cellStyle.Render(style.Render(runewidth.Truncate(value, colWidth, "…")))
}

// View renders the table with custom scrolling logic.
func (lv ListView) View() string {
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
	if end > len(lv.rows) {
		end = len(lv.rows)
		start = end - visibleRows
		if start < 0 {
			start = 0
		}
	}

	// Render visible rows
	renderedRows := make([]string, 0, visibleRows)
	for i := start; i < end; i++ {
		cells := make([]string, 0, len(lv.columns))
		for ci, value := range lv.rows[i] {
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

	visibleRows := height - 12 - 2
	if visibleRows < 3 {
		visibleRows = 3
	}
	lv.visibleRows = visibleRows
}
