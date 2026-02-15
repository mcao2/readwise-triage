package ui

import (
	"fmt"

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
	StateEditing
	StateBatchEditing
	StateConfirming
	StateUpdating
	StateDone
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
	case StateEditing:
		return "Editing"
	case StateBatchEditing:
		return "Batch Editing"
	case StateConfirming:
		return "Confirming"
	case StateUpdating:
		return "Updating"
	case StateDone:
		return "Done"
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

	// Config
	cfg *config.Config
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
	// Load config
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{DefaultDaysAgo: 7}
	}

	// Determine theme
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

	// Determine mode (default to true if not set)
	useLLM := cfg.UseLLMTriage
	if !useLLM && cfg.Theme == "" {
		// First run, default to true
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
		m.listView.SetItems(msg.Items)
		m.state = StateReviewing

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
	case StateEditing:
		return m.editingView()
	case StateBatchEditing:
		return m.batchEditingView()
	case StateConfirming:
		return m.confirmingView()
	case StateUpdating:
		return m.updatingView()
	case StateDone:
		return m.doneView()
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
	case StateEditing:
		return m.handleEditingKeys(msg)
	case StateBatchEditing:
		return m.handleBatchEditingKeys(msg)
	case StateConfirming:
		return m.handleConfirmingKeys(msg)
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
				DaysAgo:  m.cfg.DefaultDaysAgo,
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
			// TODO: Implement actual LLM triage with Perplexity API
			// For now, simulate processing delay
			return StateChangeMsg{State: StateReviewing}
		},
	)
}

func (m Model) handleReviewingKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case keyMatches(msg, m.keys.Up):
		m.listView.MoveCursor(-1)
		m.cursor = m.listView.Cursor()
	case keyMatches(msg, m.keys.Down):
		m.listView.MoveCursor(1)
		m.cursor = m.listView.Cursor()
	case keyMatches(msg, m.keys.Enter):
		if item := m.listView.GetItem(m.listView.Cursor()); item != nil {
			m.editingItem = item
			m.state = StateEditing
		}
	case keyMatches(msg, m.keys.Select):
		m.listView.ToggleSelection()
		m.cursor = m.listView.Cursor()
	case msg.String() == "b":
		if len(m.listView.GetSelected()) > 0 {
			m.state = StateBatchEditing
		}
	case msg.String() == "p":
		// Trigger LLM triage for all items
		if m.cfg != nil && m.cfg.PerplexityAPIKey != "" {
			return m, m.startTriaging()
		}
	case keyMatches(msg, m.keys.Back):
		return m, tea.Quit
	}

	if item := m.listView.GetItem(m.listView.Cursor()); item != nil {
		switch msg.String() {
		case "r":
			item.Action = "read_now"
			m.listView.SetItems(m.items)
		case "l":
			item.Action = "later"
			m.listView.SetItems(m.items)
		case "a":
			item.Action = "archive"
			m.listView.SetItems(m.items)
		case "d":
			item.Action = "delete"
			m.listView.SetItems(m.items)
		case "1":
			item.Priority = "high"
			m.listView.SetItems(m.items)
		case "2":
			item.Priority = "medium"
			m.listView.SetItems(m.items)
		case "3":
			item.Priority = "low"
			m.listView.SetItems(m.items)
		}
	}

	return m, nil
}

func (m Model) handleEditingKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case keyMatches(msg, m.keys.Back), keyMatches(msg, m.keys.Quit):
		m.editingItem = nil
		m.state = StateReviewing
	}
	return m, nil
}

func (m Model) handleBatchEditingKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case keyMatches(msg, m.keys.Back), keyMatches(msg, m.keys.Quit):
		m.state = StateReviewing
	}
	return m, nil
}

func (m Model) handleConfirmingKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case keyMatches(msg, m.keys.Enter):
		m.state = StateUpdating
	case keyMatches(msg, m.keys.Back):
		m.state = StateReviewing
	}
	return m, nil
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
	help := m.styles.Help.Render("j/k: navigate • x: select • r/l/a/d: action • 1/2/3: priority • p: AI triage • enter: edit • b: batch • q: quit")

	return lipgloss.JoinVertical(lipgloss.Left, title, "", list, "", count, help)
}

func (m Model) editingView() string {
	if m.editingItem == nil {
		return "No item selected"
	}

	title := m.styles.Title.Render("Edit Item")
	itemTitle := m.styles.Highlight.Render(m.editingItem.Title)
	help := m.styles.Help.Render("Press ESC to go back")

	return lipgloss.JoinVertical(lipgloss.Left, title, "", itemTitle, "", help)
}

func (m Model) batchEditingView() string {
	selected := m.listView.GetSelected()
	title := m.styles.Title.Render("Batch Edit")
	count := m.styles.Normal.Render(fmt.Sprintf("Editing %d selected items", len(selected)))
	help := m.styles.Help.Render("Press ESC to go back")

	return lipgloss.JoinVertical(lipgloss.Left, title, "", count, "", help)
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

func keyMatches(msg tea.KeyMsg, target key.Binding) bool {
	for _, k := range target.Keys() {
		if msg.String() == k {
			return true
		}
	}
	return false
}
