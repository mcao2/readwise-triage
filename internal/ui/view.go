package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) fetchingView() string {
	spinnerView := m.spinner.View()
	status := fmt.Sprintf("%s Loading from Readwise...", spinnerView)

	fetchTitle := "Fetching Inbox Items"

	content := m.styles.Border.Render(
		lipgloss.JoinVertical(lipgloss.Center,
			m.styles.Title.Render(fetchTitle),
			"",
			m.styles.Normal.Render(status),
		),
	)

	help := m.renderHelpLine([]helpEntry{{"q", "cancel"}})

	return lipgloss.JoinVertical(lipgloss.Center, "", content, "", help)
}

func (m *Model) reviewingView() string {
	// Header bar
	headerLeft := m.styles.HelpKey.Render("Readwise Triage")
	countText := m.styles.HelpDesc.Render(fmt.Sprintf("%d/%d", m.cursor+1, len(m.items)))
	if m.batchMode {
		selectedCount := len(m.listView.GetSelected())
		countText += m.styles.Highlight.Render(fmt.Sprintf("  ● %d selected", selectedCount))
	}
	headerGap := ""
	if m.width > 0 {
		gap := m.width - lipgloss.Width(headerLeft) - lipgloss.Width(countText) - 4
		if gap > 0 {
			headerGap = strings.Repeat(" ", gap)
		}
	}
	header := m.styles.HeaderBar.Width(m.width - 1).Render(headerLeft + headerGap + countText)

	// Table
	var list string
	if len(m.items) == 0 {
		list = m.styles.Normal.Render("  No items to review")
	} else {
		list = m.listView.View()
	}

	// Detail pane (simple padded text, no border)
	detail := ""
	if !m.editingTags && len(m.items) > 0 {
		detailContent := m.listView.DetailView(m.width, m.styles)
		if detailContent != "" {
			divW := m.width - 1
			if divW < 1 {
				divW = 1
			}
			divider := m.styles.HelpSep.Render(strings.Repeat("─", divW))
			detail = divider + "\n" + detailContent
		}
	}

	// Status message
	var statusLine string
	if m.statusMessage != "" {
		if m.messageType == "error" {
			statusLine = m.styles.Error.Render("  ✗ " + m.statusMessage)
		} else if m.messageType == "success" {
			statusLine = m.styles.Success.Render("  ✓ " + m.statusMessage)
		} else {
			statusLine = m.styles.Help.Render("  " + m.statusMessage)
		}
	}

	// Help overlay or footer (hidden during tag editing)
	var footer string
	if !m.editingTags {
		if m.showHelp {
			footer = m.renderFullHelp()
		} else {
			footer = m.renderReviewFooter()
		}
	}

	parts := []string{header, list}
	if detail != "" {
		parts = append(parts, detail)
	}
	if statusLine != "" {
		parts = append(parts, statusLine)
	}
	if footer != "" {
		parts = append(parts, footer)
	}

	content := strings.Join(parts, "\n")

	// Tag editing popup — overlaid on top of the review view
	if m.editingTags && m.height > 0 {
		runes := []rune(m.tagsInput)
		before := string(runes[:m.tagsCursor])
		after := string(runes[m.tagsCursor:])
		inputLine := fmt.Sprintf("tags: %s▌%s", before, after)
		helpLine := m.renderHelpLine([]helpEntry{{"enter", "confirm"}, {"esc", "cancel"}, {"←/→", "move"}, {"opt+←/→", "word"}})
		popup := m.styles.Card.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				m.styles.Title.Render("Edit Tags"),
				"",
				m.styles.Normal.Render(inputLine),
				"",
				helpLine,
			),
		)

		// Split both the background and popup into lines
		bgLines := strings.Split(content, "\n")
		for len(bgLines) < m.height {
			bgLines = append(bgLines, "")
		}
		bgLines = bgLines[:m.height]

		popupLines := strings.Split(popup, "\n")
		popupH := len(popupLines)

		w := m.width - 1
		if w < 1 {
			w = 1
		}

		// Center the popup lines horizontally and stamp them over the background
		startY := (m.height - popupH) / 2
		for i, pLine := range popupLines {
			row := startY + i
			if row >= 0 && row < m.height {
				bgLines[row] = lipgloss.PlaceHorizontal(w, lipgloss.Center, pLine)
			}
		}
		content = strings.Join(bgLines, "\n")
	}

	// Confirm popup overlay
	if m.confirming && m.height > 0 {
		popup := m.styles.Card.Render(
			lipgloss.JoinVertical(lipgloss.Center,
				m.styles.Title.Render("Push to Readwise?"),
				"",
				m.styles.Normal.Render(fmt.Sprintf("%d items to update", m.countUpdatable())),
				"",
				m.renderHelpLine([]helpEntry{{"y", "confirm"}, {"n", "cancel"}}),
			),
		)

		bgLines := strings.Split(content, "\n")
		for len(bgLines) < m.height {
			bgLines = append(bgLines, "")
		}
		bgLines = bgLines[:m.height]

		popupLines := strings.Split(popup, "\n")
		popupH := len(popupLines)
		w := m.width - 1
		if w < 1 {
			w = 1
		}
		startY := (m.height - popupH) / 2
		for i, pLine := range popupLines {
			row := startY + i
			if row >= 0 && row < m.height {
				bgLines[row] = lipgloss.PlaceHorizontal(w, lipgloss.Center, pLine)
			}
		}
		content = strings.Join(bgLines, "\n")
	}

	// Pad output to exactly m.height lines so the alternate screen buffer
	// repaints cleanly and doesn't leave stale content from previous frames.
	if m.height > 0 {
		rendered := strings.Split(content, "\n")
		for len(rendered) < m.height {
			rendered = append(rendered, "")
		}
		return strings.Join(rendered[:m.height], "\n")
	}
	return content
}

// Help rendering

type helpEntry struct {
	key  string
	desc string
}

func (m *Model) renderHelpLine(entries []helpEntry) string {
	var parts []string
	sep := m.styles.HelpSep.Render(" · ")
	for _, e := range entries {
		parts = append(parts, m.styles.HelpKey.Render(e.key)+" "+m.styles.HelpDesc.Render(e.desc))
	}
	return strings.Join(parts, sep)
}

func (m *Model) renderReviewFooter() string {
	var line1, line2 []helpEntry

	if m.batchMode {
		line1 = []helpEntry{
			{"j/k", "navigate"},
			{"x", "deselect"},
			{"r l a d", "action"},
		}
	} else {
		line1 = []helpEntry{
			{"j/k", "navigate"},
			{"x", "select"},
			{"r l a d", "action"},
		}
	}

	line2 = []helpEntry{
		{"enter", "tags"},
		{"T", "auto-triage"},
		{"o", "open"},
		{"f", "more"},
		{"R", "refresh"},
		{"u", "update"},
		{"?", "help"},
		{"q", "quit"},
	}

	footer := m.styles.FooterBar.Width(m.width - 1).Render(
		m.renderHelpLine(line1) + "\n" + m.renderHelpLine(line2),
	)
	return footer
}

func (m *Model) renderFullHelp() string {
	sections := []struct {
		title   string
		entries []helpEntry
	}{
		{"Navigation", []helpEntry{
			{"j / ↓", "move down"},
			{"k / ↑", "move up"},
			{"x / space", "toggle select"},
		}},
		{"Triage Actions", []helpEntry{
			{"r", "read now"},
			{"l", "later"},
			{"a", "archive"},
			{"d", "archive"},
		}},
		{"Operations", []helpEntry{
			{"enter", "edit tags"},
			{"T", "auto-triage with LLM"},
			{"o", "open URL in browser"},
			{"u", "update Readwise"},
			{"f", "fetch more (+7 days)"},
			{"R", "refresh from Readwise"},
		}},
		{"General", []helpEntry{
			{"?", "toggle this help"},
			{"q / ctrl+c", "quit"},
		}},
	}

	var lines []string
	for _, sec := range sections {
		lines = append(lines, m.styles.HelpKey.Render("  "+sec.title))
		for _, e := range sec.entries {
			lines = append(lines, fmt.Sprintf("    %s  %s",
				m.styles.HelpKey.Render(fmt.Sprintf("%-12s", e.key)),
				m.styles.HelpDesc.Render(e.desc),
			))
		}
	}

	return m.styles.FooterBar.Width(m.width - 1).Render(strings.Join(lines, "\n"))
}
