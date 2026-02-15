package ui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mcao2/readwise-triage/internal/config"
	"github.com/mcao2/readwise-triage/internal/readwise"
)

type State int

const (
	StateConfig State = iota
	StateFetching
	StateTriaging
	StateReviewing
	StateConfirming
	StateUpdating
	StateDone
	StateMessage
)

func (s State) String() string {
	switch s {
	case StateConfig:
		return "Config"
	case StateFetching:
		return "Fetching"
	case StateTriaging:
		return "Triaging"
	case StateReviewing:
		return "Reviewing"
	case StateConfirming:
		return "Confirming"
	case StateUpdating:
		return "Updating"
	case StateDone:
		return "Done"
	case StateMessage:
		return "Message"
	default:
		return "Unknown"
	}
}

type Model struct {
	state  State
	width  int
	height int
	styles Styles
	keys   KeyMap

	useLLMTriage bool
	themeIndex   int
	showHelp     bool

	items  []Item
	cursor int

	listView ListView
	spinner  spinner.Model
	progress progress.Model

	updateProgress float64
	statusMessage  string
	messageType    string
	batchMode      bool

	cfg         *config.Config
	triageStore *config.TriageStore

	fetchLookback int
}

type Item struct {
	ID          string
	Title       string
	Action      string
	Priority    string
	URL         string
	Summary     string
	Category    string
	Source      string
	WordCount   int
	ReadingTime string
	Tags        []string
}

func NewModel() *Model {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{DefaultDaysAgo: 7}
	}

	triageStore, err := config.LoadTriageStore()
	if err != nil {
		triageStore = &config.TriageStore{Items: make(map[string]config.TriageEntry)}
	}

	themeNames := GetThemeNames()
	themeIndex := -1
	themeName := cfg.Theme

	for i, name := range themeNames {
		if name == themeName {
			themeIndex = i
			break
		}
	}

	if themeIndex == -1 {
		themeName = "default"
		for i, name := range themeNames {
			if name == themeName {
				themeIndex = i
				break
			}
		}
	}

	if themeIndex == -1 && len(themeNames) > 0 {
		themeIndex = 0
		themeName = themeNames[0]
	}

	useLLM := cfg.UseLLMTriage
	if !useLLM && cfg.Theme == "" {
		useLLM = true
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(Themes[themeName].Primary))

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithoutPercentage(),
	)

	m := &Model{
		state:         StateConfig,
		useLLMTriage:  useLLM,
		styles:        NewStyles(Themes[themeName]),
		keys:          DefaultKeyMap(),
		themeIndex:    themeIndex,
		items:         []Item{},
		cursor:        0,
		spinner:       s,
		progress:      p,
		cfg:           cfg,
		triageStore:   triageStore,
		fetchLookback: cfg.DefaultDaysAgo,
	}
	m.listView = NewListView(80, 24)
	m.listView.UpdateTableStyles(Themes[themeName])
	return m
}

func (m *Model) cycleTheme() {
	themeNames := GetThemeNames()
	m.themeIndex = (m.themeIndex + 1) % len(themeNames)
	newTheme := themeNames[m.themeIndex]
	m.styles = NewStyles(Themes[newTheme])
	m.listView.UpdateTableStyles(Themes[newTheme])
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(Themes[newTheme].Primary))

	if m.cfg != nil {
		m.cfg.Theme = newTheme
		_ = m.cfg.Save()
	}
}

func (m *Model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.listView.SetWidthHeight(msg.Width, msg.Height)
		m.progress.Width = msg.Width - 8

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case StateChangeMsg:
		m.state = msg.State

	case ProgressMsg:
		m.updateProgress = msg.Progress
		m.statusMessage = msg.Message
		cmd := m.progress.SetPercent(msg.Progress)
		return m, tea.Batch(cmd, m.waitForUpdateProgress(msg.Channel, msg.Success, msg.Failed))

	case ItemsLoadedMsg:
		m.items = msg.Items
		m.applySavedTriages()
		m.listView.SetItems(m.items)
		m.statusMessage = fmt.Sprintf("Loaded %d items from the last %d days", len(m.items), m.fetchLookback)
		m.state = StateReviewing

	case UpdateFinishedMsg:
		m.statusMessage = fmt.Sprintf("Successfully updated %d items (%d failed)", msg.Success, msg.Failed)
		m.state = StateDone

	case ErrorMsg:
		m.statusMessage = msg.Error.Error()
		m.state = StateConfig
	}

	return m, nil
}

func (m *Model) View() string {
	switch m.state {
	case StateConfig:
		return m.configView()
	case StateFetching:
		return m.fetchingView()
	case StateTriaging:
		return m.triagingView()
	case StateReviewing:
		return m.reviewingView()
	case StateConfirming:
		return m.confirmingView()
	case StateUpdating:
		return m.updatingView()
	case StateDone:
		return m.doneView()
	case StateMessage:
		return m.messageView()
	default:
		return "Unknown state"
	}
}

func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case StateDone:
		return m.handleDoneKeys(msg)
	case StateMessage:
		return m.handleMessageKeys(msg)
	}

	switch {
	case keyMatches(msg, m.keys.Quit):
		return m, tea.Quit
	case keyMatches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
		return m, nil
	}

	switch m.state {
	case StateConfig:
		return m.handleConfigKeys(msg)
	case StateReviewing:
		return m.handleReviewingKeys(msg)
	case StateConfirming:
		return m.handleConfirmingKeys(msg)
	}

	return m, nil
}

type StateChangeMsg struct {
	State State
}

type ProgressMsg struct {
	Progress float64
	Message  string
	Success  int
	Failed   int
	Channel  chan readwise.BatchUpdateProgress
}

type ItemsLoadedMsg struct {
	Items []Item
}

type ErrorMsg struct {
	Error error
}

type UpdateFinishedMsg struct {
	Success int
	Failed  int
}

func (m *Model) handleConfigKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case keyMatches(msg, m.keys.Enter):
		return m, m.startFetching()
	case keyMatches(msg, m.keys.ToggleMode):
		m.useLLMTriage = !m.useLLMTriage
		if m.cfg != nil {
			m.cfg.UseLLMTriage = m.useLLMTriage
			_ = m.cfg.Save()
		}
	case keyMatches(msg, m.keys.CycleTheme):
		m.cycleTheme()
	}
	return m, nil
}

func (m *Model) startFetching() tea.Cmd {
	m.state = StateFetching
	m.statusMessage = "Loading from Readwise..."

	return func() tea.Msg {
		if m.cfg == nil || m.cfg.ReadwiseToken == "" {
			return ErrorMsg{Error: fmt.Errorf("READWISE_TOKEN not configured. Set it via environment variable or config file")}
		}

		client, err := readwise.NewClient(m.cfg.ReadwiseToken)
		if err != nil {
			return ErrorMsg{Error: err}
		}

		opts := readwise.FetchOptions{
			DaysAgo:  m.fetchLookback,
			Location: "new",
		}
		items, err := client.GetInboxItems(opts)
		if err != nil {
			return ErrorMsg{Error: err}
		}

		uiItems := make([]Item, len(items))
		for i, item := range items {
			uiItems[i] = Item{
				ID:          item.ID,
				Title:       item.Title,
				Action:      "",
				Priority:    "",
				URL:         item.URL,
				Summary:     item.Summary,
				Category:    item.Category,
				Source:      item.Source,
				WordCount:   item.WordCount,
				ReadingTime: item.ReadingTime,
			}
		}

		return ItemsLoadedMsg{Items: uiItems}
	}
}

func (m *Model) startTriaging() tea.Cmd {
	m.state = StateTriaging
	return func() tea.Msg {
		return ErrorMsg{Error: fmt.Errorf("LLM triage not yet implemented")}
	}
}

func (m *Model) startUpdating() tea.Cmd {
	if m.cfg == nil || m.cfg.ReadwiseToken == "" {
		return func() tea.Msg {
			return ErrorMsg{Error: fmt.Errorf("READWISE_TOKEN not configured")}
		}
	}

	selectedIndices := m.listView.GetSelected()
	useSelection := len(selectedIndices) > 0

	var updates []readwise.UpdateRequest
	for i, item := range m.items {
		if useSelection {
			isSelected := false
			for _, idx := range selectedIndices {
				if idx == i {
					isSelected = true
					break
				}
			}
			if !isSelected {
				continue
			}
		}

		if item.Action != "" {
			update := readwise.UpdateRequest{
				DocumentID: item.ID,
			}

			switch item.Action {
			case "read_now":
				update.Tags = []string{"read_now"}
			case "later":
				update.Location = "later"
			case "archive", "delete":
				update.Location = "archive"
			case "needs_review":
				update.Tags = []string{"needs_review"}
			}

			if item.Priority != "" {
				update.Tags = append(update.Tags, "priority:"+item.Priority)
			}

			// Add LLM-suggested tags
			if len(item.Tags) > 0 {
				update.Tags = append(update.Tags, item.Tags...)
			}

			updates = append(updates, update)
		}
	}

	if len(updates) == 0 {
		return func() tea.Msg {
			return UpdateFinishedMsg{Success: 0, Failed: 0}
		}
	}

	m.state = StateUpdating
	m.updateProgress = 0
	m.statusMessage = "Preparing updates..."

	progressChan := make(chan readwise.BatchUpdateProgress)

	go func() {
		client, err := readwise.NewClient(m.cfg.ReadwiseToken)
		if err == nil {
			client.BatchUpdate(updates, progressChan)
		}
		close(progressChan)
	}()

	return m.waitForUpdateProgress(progressChan, 0, 0)
}

func (m *Model) waitForUpdateProgress(ch chan readwise.BatchUpdateProgress, success, failed int) tea.Cmd {
	return func() tea.Msg {
		progress, ok := <-ch
		if !ok {
			return UpdateFinishedMsg{Success: success, Failed: failed}
		}

		newSuccess := success
		newFailed := failed
		if progress.Success {
			newSuccess++
		} else {
			newFailed++
		}

		return ProgressMsg{
			Progress: float64(progress.Current) / float64(progress.Total),
			Message:  fmt.Sprintf("Updated %d/%d items", progress.Current, progress.Total),
			Success:  newSuccess,
			Failed:   newFailed,
			Channel:  ch,
		}
	}
}

func (m *Model) handleReviewingKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case keyMatches(msg, m.keys.Up), keyMatches(msg, m.keys.Down):
		// Forward navigation keys to the table so it handles scrolling/viewport
		m.listView.UpdateTable(msg)
		m.cursor = m.listView.SyncCursor()
		return m, nil
	case keyMatches(msg, m.keys.Open):
		selected := m.listView.GetSelected()
		if len(selected) > 0 {
			for _, idx := range selected {
				if item := m.listView.GetItem(idx); item != nil {
					_ = openURL(item.URL)
				}
			}
		} else if item := m.listView.GetItem(m.listView.Cursor()); item != nil {
			if err := openURL(item.URL); err != nil {
				m.statusMessage = fmt.Sprintf("Failed to open URL: %v", err)
				m.messageType = "error"
				m.state = StateMessage
			}
		}
		return m, nil
	case keyMatches(msg, m.keys.Select):
		m.listView.ToggleSelection()
		m.cursor = m.listView.Cursor()
		m.batchMode = len(m.listView.GetSelected()) > 0
		return m, nil
	case msg.String() == "e":
		if err := m.ExportItemsToClipboard(); err != nil {
			m.statusMessage = fmt.Sprintf("Export failed: %v", err)
			m.messageType = "error"
		} else {
			m.statusMessage = "Items exported to clipboard! Paste to Perplexity."
			m.messageType = "success"
		}
		m.state = StateMessage
		return m, nil
	case msg.String() == "i":
		applied, err := m.ImportTriageResultsFromClipboard()
		if err != nil {
			m.statusMessage = fmt.Sprintf("Import failed: %v", err)
			m.messageType = "error"
		} else {
			if m.statusMessage == "" {
				m.statusMessage = fmt.Sprintf("Applied triage results to %d items", applied)
			}
			m.messageType = "success"
		}
		m.state = StateMessage
		return m, nil
	case keyMatches(msg, m.keys.Update):
		m.state = StateConfirming
		return m, nil
	case keyMatches(msg, m.keys.FetchMore):
		m.fetchLookback += 7
		if m.cfg != nil {
			m.cfg.DefaultDaysAgo = m.fetchLookback
			_ = m.cfg.Save()
		}
		return m, m.startFetching()
	case keyMatches(msg, m.keys.Refresh):
		return m, m.startFetching()
	case keyMatches(msg, m.keys.Back):
		return m, tea.Quit
	}

	if m.batchMode {
		switch msg.String() {
		case "r":
			m.applyBatchAction("read_now")
		case "l":
			m.applyBatchAction("later")
		case "a":
			m.applyBatchAction("archive")
		case "d":
			m.applyBatchAction("delete")
		case "n":
			m.applyBatchAction("needs_review")
		case "1":
			m.applyBatchPriority("high")
		case "2":
			m.applyBatchPriority("medium")
		case "3":
			m.applyBatchPriority("low")
		}
		return m, nil
	}

	if item := m.listView.GetItem(m.listView.Cursor()); item != nil {
		switch msg.String() {
		case "r":
			m.setItemAction(item, "read_now")
		case "l":
			m.setItemAction(item, "later")
		case "a":
			m.setItemAction(item, "archive")
		case "d":
			m.setItemAction(item, "delete")
		case "n":
			m.setItemAction(item, "needs_review")
		case "1":
			m.setItemPriority(item, "high")
		case "2":
			m.setItemPriority(item, "medium")
		case "3":
			m.setItemPriority(item, "low")
		}
	}

	return m, nil
}

func (m *Model) applyBatchAction(action string) {
	selected := m.listView.GetSelected()
	for _, idx := range selected {
		if idx >= 0 && idx < len(m.items) {
			m.items[idx].Action = action
			m.saveTriage(m.items[idx].ID, m.items[idx].Action, m.items[idx].Priority)
		}
	}
	m.listView.SetItems(m.items)
}

func (m *Model) applyBatchPriority(priority string) {
	selected := m.listView.GetSelected()
	for _, idx := range selected {
		if idx >= 0 && idx < len(m.items) {
			m.items[idx].Priority = priority
			m.saveTriage(m.items[idx].ID, m.items[idx].Action, m.items[idx].Priority)
		}
	}
	m.listView.SetItems(m.items)
}

func (m *Model) setItemAction(item *Item, action string) {
	item.Action = action
	m.saveTriage(item.ID, item.Action, item.Priority)
	m.listView.SetItems(m.items)
}

func (m *Model) setItemPriority(item *Item, priority string) {
	item.Priority = priority
	m.saveTriage(item.ID, item.Action, item.Priority)
	m.listView.SetItems(m.items)
}

func (m *Model) handleConfirmingKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		return m, m.startUpdating()
	case "n", "N", "esc":
		m.state = StateReviewing
	}
	return m, nil
}

func (m *Model) handleDoneKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.state = StateConfig
	m.statusMessage = ""
	return m, nil
}

func (m *Model) handleMessageKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.state = StateReviewing
	m.statusMessage = ""
	return m, nil
}

func (m *Model) applySavedTriages() {
	if m.triageStore == nil {
		return
	}
	for i := range m.items {
		if entry, ok := m.triageStore.GetItem(m.items[i].ID); ok {
			m.items[i].Action = entry.Action
			m.items[i].Priority = entry.Priority
		}
	}
}

func (m *Model) saveTriage(id, action, priority string) {
	if m.triageStore == nil {
		return
	}
	m.triageStore.SetItem(id, action, priority, "manual")
	_ = m.triageStore.Save()
}

func (m *Model) saveLLMTriage(id, action, priority string) {
	if m.triageStore == nil {
		return
	}
	m.triageStore.SetItem(id, action, priority, "llm")
	_ = m.triageStore.Save()
}

func (m *Model) configView() string {
	// Styled title
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(m.styles.theme.Primary)).
		Render("  Readwise Triage")

	// Mode indicator
	var modeIcon, modeLabel string
	if m.useLLMTriage {
		modeIcon = "🤖"
		modeLabel = "LLM Auto-Triage (Perplexity)"
	} else {
		modeIcon = "✋"
		modeLabel = "Manual Triage"
	}
	modeLine := fmt.Sprintf("  %s  %s", modeIcon, m.styles.Normal.Render(modeLabel))

	// Theme indicator
	themeName := m.cfg.Theme
	if themeName == "" {
		themeName = "default"
	}
	themeLine := fmt.Sprintf("  🎨  %s", m.styles.Normal.Render("Theme: "+themeName))

	// Days lookback
	daysLine := fmt.Sprintf("  📅  %s", m.styles.Normal.Render(fmt.Sprintf("Fetch last %d days", m.fetchLookback)))

	content := lipgloss.JoinVertical(lipgloss.Left,
		"",
		title,
		"",
		modeLine,
		themeLine,
		daysLine,
		"",
	)

	// Error display
	if m.statusMessage != "" {
		errLine := m.styles.Error.Render("  ⚠  " + m.statusMessage)
		content = lipgloss.JoinVertical(lipgloss.Left, content, errLine, "")
	}

	// Help
	help := m.renderHelpLine([]helpEntry{
		{"enter", "start"},
		{"m", "mode"},
		{"t", "theme"},
		{"q", "quit"},
	})

	card := m.styles.Card.Render(content)

	return lipgloss.JoinVertical(lipgloss.Center,
		"",
		card,
		"",
		help,
	)
}

func (m *Model) fetchingView() string {
	spinnerView := m.spinner.View()
	status := fmt.Sprintf("%s Loading from Readwise...", spinnerView)

	content := m.styles.Border.Render(
		lipgloss.JoinVertical(lipgloss.Center,
			m.styles.Title.Render("Fetching Inbox Items"),
			"",
			m.styles.Normal.Render(status),
		),
	)

	entries := []helpEntry{{"q", "cancel"}}
	if m.useLLMTriage {
		entries = append([]helpEntry{{"s", "skip LLM"}}, entries...)
	}
	help := m.renderHelpLine(entries)

	return lipgloss.JoinVertical(lipgloss.Center, "", content, "", help)
}

func (m *Model) triagingView() string {
	spinnerView := m.spinner.View()
	status := fmt.Sprintf("%s Processing with LLM...", spinnerView)

	content := m.styles.Border.Render(
		lipgloss.JoinVertical(lipgloss.Center,
			m.styles.Title.Render("Triaging Items"),
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
	header := m.styles.HeaderBar.Width(m.width).Render(headerLeft + headerGap + countText)

	// Table
	var list string
	if len(m.items) == 0 {
		list = m.styles.Normal.Render("  No items to review")
	} else {
		list = m.listView.View()
	}

	// Detail pane (simple padded text, no border)
	detail := ""
	if len(m.items) > 0 {
		detailContent := m.listView.DetailView(m.width, m.styles)
		if detailContent != "" {
			divider := m.styles.HelpSep.Render(strings.Repeat("─", m.width))
			detail = divider + "\n" + detailContent
		}
	}

	// Status message
	var statusLine string
	if m.statusMessage != "" {
		statusLine = m.styles.Help.Render("  " + m.statusMessage)
	}

	// Help overlay or footer
	var footer string
	if m.showHelp {
		footer = m.renderFullHelp()
	} else {
		footer = m.renderReviewFooter()
	}

	parts := []string{header, list}
	if detail != "" {
		parts = append(parts, detail)
	}
	if statusLine != "" {
		parts = append(parts, statusLine)
	}
	parts = append(parts, footer)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m *Model) confirmingView() string {
	content := m.styles.Border.Render(
		lipgloss.JoinVertical(lipgloss.Center,
			m.styles.Title.Render("Confirm Update"),
			"",
			m.styles.Normal.Render("Push changes to Readwise?"),
		),
	)

	help := m.renderHelpLine([]helpEntry{
		{"y", "confirm"},
		{"n", "cancel"},
	})

	return lipgloss.JoinVertical(lipgloss.Center, "", content, "", help)
}

func (m *Model) updatingView() string {
	spinnerView := m.spinner.View()
	progressBar := m.progress.View()
	pctText := fmt.Sprintf("%.0f%%", m.updateProgress*100)

	content := m.styles.Border.Render(
		lipgloss.JoinVertical(lipgloss.Center,
			m.styles.Title.Render("Updating Readwise"),
			"",
			fmt.Sprintf("%s %s  %s", spinnerView, m.styles.Normal.Render(m.statusMessage), m.styles.Help.Render(pctText)),
			"",
			progressBar,
		),
	)

	return lipgloss.JoinVertical(lipgloss.Center, "", content)
}

func (m *Model) doneView() string {
	content := m.styles.Border.Render(
		lipgloss.JoinVertical(lipgloss.Center,
			m.styles.Success.Render("✓ Complete"),
			"",
			m.styles.Normal.Render(m.statusMessage),
		),
	)

	help := m.renderHelpLine([]helpEntry{{"any key", "continue"}})
	return lipgloss.JoinVertical(lipgloss.Center, "", content, "", help)
}

func (m *Model) messageView() string {
	var icon, title string
	var titleStyle lipgloss.Style

	if m.messageType == "error" {
		icon = "✗"
		title = "Error"
		titleStyle = m.styles.Error
	} else {
		icon = "✓"
		title = "Success"
		titleStyle = m.styles.Success
	}

	content := m.styles.Border.Render(
		lipgloss.JoinVertical(lipgloss.Center,
			titleStyle.Render(icon+" "+title),
			"",
			m.styles.Normal.Render(m.statusMessage),
		),
	)

	help := m.renderHelpLine([]helpEntry{{"any key", "continue"}})
	return lipgloss.JoinVertical(lipgloss.Center, "", content, "", help)
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
			{"r l a d n", "action"},
			{"1 2 3", "priority"},
		}
	} else {
		line1 = []helpEntry{
			{"j/k", "navigate"},
			{"x", "select"},
			{"r l a d n", "action"},
			{"1 2 3", "priority"},
		}
	}

	line2 = []helpEntry{
		{"e", "export"},
		{"i", "import"},
		{"o", "open"},
		{"f", "more"},
		{"R", "refresh"},
		{"u", "update"},
		{"?", "help"},
		{"q", "quit"},
	}

	footer := m.styles.FooterBar.Width(m.width).Render(
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
			{"d", "delete"},
			{"n", "needs review"},
		}},
		{"Priority", []helpEntry{
			{"1", "high"},
			{"2", "medium"},
			{"3", "low"},
		}},
		{"Operations", []helpEntry{
			{"e", "export to clipboard"},
			{"i", "import from clipboard"},
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

	return m.styles.FooterBar.Width(m.width).Render(strings.Join(lines, "\n"))
}

func keyMatches(msg tea.KeyMsg, target key.Binding) bool {
	for _, k := range target.Keys() {
		if msg.String() == k {
			return true
		}
	}
	return false
}

func openURL(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	return exec.Command(cmd, args...).Start()
}
