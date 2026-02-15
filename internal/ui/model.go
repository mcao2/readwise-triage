package ui

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mcao2/readwise-triage/internal/config"
	"github.com/mcao2/readwise-triage/internal/readwise"
)

// State represents the current application state
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

// Model is the main Bubble Tea model
type Model struct {
	state  State
	width  int
	height int
	styles Styles
	keys   KeyMap

	useLLMTriage bool
	themeIndex   int

	// Data
	items         []Item
	selectedIndex int
	cursor        int
	selected      map[int]bool

	listView ListView

	// Editing
	editingItem *Item

	// Progress
	progress      float64
	statusMessage string
	messageType   string // "error" or "success"
	batchMode     bool   // true when items are selected for batch editing

	// Config
	cfg         *config.Config
	triageStore *config.TriageStore

	fetchLookback int
}

// Item represents a displayable item in the list
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
}

// NewModel creates a new UI model
func NewModel() Model {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{DefaultDaysAgo: 7}
	}

	triageStore, err := config.LoadTriageStore()
	if err != nil {
		triageStore = &config.TriageStore{Items: make(map[string]config.TriageEntry)}
	}

	themeIndex := 0
	themeName := cfg.Theme
	if themeName == "" {
		themeName = "default"
	}
	for i, name := range GetThemeNames() {
		if name == themeName {
			themeIndex = i
			break
		}
	}

	useLLM := cfg.UseLLMTriage
	if !useLLM && cfg.Theme == "" {
		useLLM = true
	}

	m := Model{
		state:         StateConfig,
		useLLMTriage:  useLLM,
		styles:        NewStyles(Themes[themeName]),
		keys:          DefaultKeyMap(),
		selectedIndex: 0,
		cursor:        0,
		selected:      make(map[int]bool),
		themeIndex:    themeIndex,
		cfg:           cfg,
		triageStore:   triageStore,
		fetchLookback: cfg.DefaultDaysAgo,
	}
	m.listView = NewListView(80, 24)
	return m
}

func (m *Model) cycleTheme() {
	themeNames := GetThemeNames()
	m.themeIndex = (m.themeIndex + 1) % len(themeNames)
	newTheme := themeNames[m.themeIndex]
	m.styles = NewStyles(Themes[newTheme])

	// Save to config
	if m.cfg != nil {
		m.cfg.Theme = newTheme
		_ = m.cfg.Save()
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.listView.SetWidthHeight(msg.Width, msg.Height)

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case StateChangeMsg:
		m.state = msg.State

	case ProgressMsg:
		m.progress = msg.Progress
		m.statusMessage = msg.Message

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

// View renders the current view
func (m Model) View() string {
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

// handleKeyPress handles keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keys
	switch {
	case keyMatches(msg, m.keys.Quit):
		return m, tea.Quit
	case keyMatches(msg, m.keys.Help):
		// Toggle help view
		return m, nil
	}

	// State-specific keys
	switch m.state {
	case StateConfig:
		return m.handleConfigKeys(msg)
	case StateReviewing:
		return m.handleReviewingKeys(msg)
	case StateConfirming:
		return m.handleConfirmingKeys(msg)
	case StateMessage:
		return m.handleMessageKeys(msg)
	}

	return m, nil
}

// StateChangeMsg triggers a state change
type StateChangeMsg struct {
	State State
}

// ProgressMsg updates progress
type ProgressMsg struct {
	Progress float64
	Message  string
}

// ItemsLoadedMsg signals items have been loaded
type ItemsLoadedMsg struct {
	Items []Item
}

// ErrorMsg signals an error occurred
type ErrorMsg struct {
	Error error
}

type UpdateFinishedMsg struct {
	Success int
	Failed  int
}

// State handlers
func (m *Model) handleConfigKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case keyMatches(msg, m.keys.Enter):
		return m, m.startFetching()
	case msg.String() == "m":
		m.useLLMTriage = !m.useLLMTriage
		if m.cfg != nil {
			m.cfg.UseLLMTriage = m.useLLMTriage
			_ = m.cfg.Save()
		}
	case msg.String() == "t":
		m.cycleTheme()
	}
	return m, nil
}

func (m Model) startFetching() tea.Cmd {
	return tea.Sequence(
		func() tea.Msg {
			return StateChangeMsg{State: StateFetching}
		},
		func() tea.Msg {
			// Check if token is configured
			if m.cfg == nil || m.cfg.ReadwiseToken == "" {
				return ErrorMsg{Error: fmt.Errorf("READWISE_TOKEN not configured. Set it via environment variable or config file")}
			}

			// Create readwise client
			client, err := readwise.NewClient(m.cfg.ReadwiseToken)
			if err != nil {
				return ErrorMsg{Error: err}
			}

			// Fetch inbox items
			opts := readwise.FetchOptions{
				DaysAgo:  m.fetchLookback,
				Location: "new",
			}
			items, err := client.GetInboxItems(opts)
			if err != nil {
				return ErrorMsg{Error: err}
			}

			// Convert to UI items
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
		},
	)
}

func (m Model) startTriaging() tea.Cmd {
	return tea.Sequence(
		func() tea.Msg {
			return StateChangeMsg{State: StateTriaging}
		},
		func() tea.Msg {
			return ErrorMsg{Error: fmt.Errorf("LLM triage not yet implemented")}
		},
	)
}

func (m Model) startUpdating() tea.Cmd {
	return tea.Sequence(
		func() tea.Msg {
			return StateChangeMsg{State: StateUpdating}
		},
		func() tea.Msg {
			if m.cfg == nil || m.cfg.ReadwiseToken == "" {
				return ErrorMsg{Error: fmt.Errorf("READWISE_TOKEN not configured")}
			}

			client, err := readwise.NewClient(m.cfg.ReadwiseToken)
			if err != nil {
				return ErrorMsg{Error: err}
			}

			var updates []readwise.UpdateRequest
			for _, item := range m.items {
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
					}

					if item.Priority != "" {
						update.Tags = append(update.Tags, "priority:"+item.Priority)
					}

					updates = append(updates, update)
				}
			}

			if len(updates) == 0 {
				return UpdateFinishedMsg{Success: 0, Failed: 0}
			}

			res, _ := client.BatchUpdate(updates, nil)
			return UpdateFinishedMsg{
				Success: res.Success,
				Failed:  res.Failed,
			}
		},
	)
}

func (m *Model) handleReviewingKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case keyMatches(msg, m.keys.Up):
		m.listView.MoveCursor(-1)
		m.cursor = m.listView.Cursor()
		return m, nil
	case keyMatches(msg, m.keys.Down):
		m.listView.MoveCursor(1)
		m.cursor = m.listView.Cursor()
		return m, nil
	case keyMatches(msg, m.keys.Open):
		if item := m.listView.GetItem(m.listView.Cursor()); item != nil {
			_ = openURL(item.URL)
		}
		return m, nil
	case msg.String() == "x" || msg.String() == " " || msg.String() == "space":
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
			m.statusMessage = fmt.Sprintf("Applied triage results to %d items", applied)
			m.messageType = "success"
		}
		m.state = StateMessage
		return m, nil
	case keyMatches(msg, m.keys.Update):
		m.state = StateConfirming
		return m, nil
	case keyMatches(msg, m.keys.FetchMore):
		m.fetchLookback += 7
		return m, m.startFetching()
	case keyMatches(msg, m.keys.Back):
		return m, tea.Quit
	}

	// Batch actions when items are selected
	if m.batchMode {
		switch msg.String() {
		case "r":
			m.applyBatchAction("read_now")
		case "l":
			m.applyBatchAction("later")
		case "a":
			m.applyBatchAction("archive")
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
			item.Action = "read_now"
			m.saveTriage(item.ID, item.Action, item.Priority)
			m.listView.SetItems(m.items)
		case "l":
			item.Action = "later"
			m.saveTriage(item.ID, item.Action, item.Priority)
			m.listView.SetItems(m.items)
		case "a":
			item.Action = "archive"
			m.saveTriage(item.ID, item.Action, item.Priority)
			m.listView.SetItems(m.items)
		case "1":
			item.Priority = "high"
			m.saveTriage(item.ID, item.Action, item.Priority)
			m.listView.SetItems(m.items)
		case "2":
			item.Priority = "medium"
			m.saveTriage(item.ID, item.Action, item.Priority)
			m.listView.SetItems(m.items)
		case "3":
			item.Priority = "low"
			m.saveTriage(item.ID, item.Action, item.Priority)
			m.listView.SetItems(m.items)
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

func (m *Model) handleConfirmingKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		return m, m.startUpdating()
	case "n", "N", "esc":
		m.state = StateReviewing
	}
	return m, nil
}

func (m *Model) handleMessageKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.state = StateReviewing
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

// View helpers
func (m Model) configView() string {
	title := m.styles.Title.Render("Readwise Triage")

	var modeText string
	if m.useLLMTriage {
		modeText = m.styles.Normal.Render("Mode: LLM Auto-Triage (Perplexity)")
	} else {
		modeText = m.styles.Normal.Render("Mode: Manual Triage")
	}

	themeName := m.cfg.Theme
	if themeName == "" {
		themeName = "default"
	}
	themeText := m.styles.Normal.Render("Theme: " + themeName)

	help := m.styles.Help.Render("Enter: start • m: toggle mode • t: change theme • q: quit")

	var errorText string
	if m.statusMessage != "" {
		errorText = m.styles.Error.Render("Error: " + m.statusMessage)
	}

	if errorText != "" {
		return lipgloss.JoinVertical(lipgloss.Center, title, "", modeText, "", themeText, "", errorText, "", help)
	}
	return lipgloss.JoinVertical(lipgloss.Center, title, "", modeText, "", themeText, "", help)
}

func (m Model) fetchingView() string {
	title := m.styles.Title.Render("Fetching Inbox Items")
	status := m.styles.Normal.Render("Loading from Readwise...")

	var helpText string
	if m.useLLMTriage {
		helpText = "s: skip LLM triage (manual mode) • q: cancel"
	} else {
		helpText = "q: cancel"
	}
	help := m.styles.Help.Render(helpText)

	return lipgloss.JoinVertical(lipgloss.Center, title, "", status, "", help)
}

func (m Model) triagingView() string {
	title := m.styles.Title.Render("Triaging Items")
	status := m.styles.Normal.Render("Processing with LLM...")
	help := m.styles.Help.Render("Press q to cancel")
	return lipgloss.JoinVertical(lipgloss.Center, title, "", status, "", help)
}

func (m Model) reviewingView() string {
	title := m.styles.Title.Render("Review Items")

	var list string
	if len(m.items) == 0 {
		list = m.styles.Normal.Render("No items to review")
	} else {
		list = m.listView.View()
	}

	count := m.styles.Help.Render(fmt.Sprintf("Item %d of %d", m.cursor+1, len(m.items)))

	var help string
	if m.batchMode {
		selectedCount := len(m.listView.GetSelected())
		batchIndicator := m.styles.Highlight.Render(fmt.Sprintf(" [BATCH: %d selected]", selectedCount))
		help = m.styles.Help.Render("j/k: navigate • x: deselect • r/l/a: batch action • 1/2/3: batch priority" + batchIndicator + " • e: export JSON • i: import triage • o: open • f: more • u: update • q: quit")
	} else {
		help = m.styles.Help.Render("j/k: navigate • x: select • r/l/a: action • 1/2/3: priority • e: export JSON • i: import triage • o: open • f: more • u: update • q: quit")
	}

	var statusText string
	if m.statusMessage != "" {
		statusText = m.styles.Normal.Render(m.statusMessage)
	}

	if statusText != "" {
		return lipgloss.JoinVertical(lipgloss.Left, title, "", list, "", count, "", statusText, help)
	}
	return lipgloss.JoinVertical(lipgloss.Left, title, "", list, "", count, help)
}

func (m Model) confirmingView() string {
	title := m.styles.Title.Render("Confirm Update")
	message := m.styles.Normal.Render("Are you sure you want to update Readwise?")
	help := m.styles.Help.Render("y: yes • n: no")
	return lipgloss.JoinVertical(lipgloss.Center, title, "", message, "", help)
}

func (m Model) updatingView() string {
	title := m.styles.Title.Render("Updating Readwise")
	progress := m.styles.Normal.Render(fmt.Sprintf("Progress: %.0f%%", m.progress*100))
	status := m.styles.Normal.Render(m.statusMessage)
	return lipgloss.JoinVertical(lipgloss.Center, title, "", progress, status)
}

func (m Model) doneView() string {
	title := m.styles.Title.Render("Complete")
	message := m.styles.Highlight.Render("All updates applied successfully!")
	help := m.styles.Help.Render("Press q to quit")
	return lipgloss.JoinVertical(lipgloss.Center, title, "", message, "", help)
}

func (m Model) messageView() string {
	var title string
	var message string

	if m.messageType == "error" {
		title = m.styles.Error.Render("Error")
		message = m.styles.Error.Render(m.statusMessage)
	} else {
		title = m.styles.Title.Render("Success")
		message = m.styles.Normal.Render(m.statusMessage)
	}

	help := m.styles.Help.Render("Press any key to continue")
	return lipgloss.JoinVertical(lipgloss.Center, title, "", message, "", help)
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
