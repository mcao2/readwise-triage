package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// State represents the current application state
type State int

const (
	StateConfig State = iota
	StateFetching
	StateTriaging
	StateReviewing
	StateEditing
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

	// Data
	items         []Item
	selectedIndex int
	cursor        int
	selected      map[int]bool

	// Editing
	editingItem *Item

	// Progress
	progress      float64
	statusMessage string
}

// Item represents a displayable item in the list
type Item struct {
	ID       string
	Title    string
	Action   string
	Priority string
	URL      string
	Summary  string
}

// NewModel creates a new UI model
func NewModel() Model {
	return Model{
		state:         StateConfig,
		useLLMTriage:  true,
		styles:        DefaultStyles(),
		keys:          DefaultKeyMap(),
		selectedIndex: 0,
		cursor:        0,
		selected:      make(map[int]bool),
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

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case StateChangeMsg:
		m.state = msg.State

	case ProgressMsg:
		m.progress = msg.Progress
		m.statusMessage = msg.Message

	case ItemsLoadedMsg:
		m.items = msg.Items
		m.state = StateReviewing
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

// State handlers
func (m Model) handleConfigKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case keyMatches(msg, m.keys.Enter):
		return m, func() tea.Msg {
			return StateChangeMsg{State: StateFetching}
		}
	case msg.String() == "m":
		m.useLLMTriage = !m.useLLMTriage
	}
	return m, nil
}

func (m Model) handleReviewingKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case keyMatches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case keyMatches(msg, m.keys.Down):
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
	case keyMatches(msg, m.keys.Enter):
		if len(m.items) > 0 && m.cursor < len(m.items) {
			m.editingItem = &m.items[m.cursor]
			m.state = StateEditing
		}
	case keyMatches(msg, m.keys.Select):
		m.selected[m.cursor] = !m.selected[m.cursor]
	case keyMatches(msg, m.keys.Back):
		return m, tea.Quit
	}

	if len(m.items) > 0 && m.cursor < len(m.items) {
		switch msg.String() {
		case "r":
			m.items[m.cursor].Action = "read_now"
		case "l":
			m.items[m.cursor].Action = "later"
		case "a":
			m.items[m.cursor].Action = "archive"
		case "d":
			m.items[m.cursor].Action = "delete"
		case "1":
			m.items[m.cursor].Priority = "high"
		case "2":
			m.items[m.cursor].Priority = "medium"
		case "3":
			m.items[m.cursor].Priority = "low"
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
	title := m.styles.Title.Render("Readwise TUI")

	var modeText string
	if m.useLLMTriage {
		modeText = m.styles.Normal.Render("Mode: LLM Auto-Triage (Perplexity)")
	} else {
		modeText = m.styles.Normal.Render("Mode: Manual Triage")
	}

	help := m.styles.Help.Render("Enter: start • m: toggle mode • q: quit")
	return lipgloss.JoinVertical(lipgloss.Center, title, "", modeText, "", help)
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
		var items []string
		for i, item := range m.items {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}
			selected := " "
			if m.selected[i] {
				selected = "x"
			}
			action := actionIcon(item.Action)
			line := m.styles.Normal.Render(cursor + " [" + selected + "] " + action + " " + item.Title)
			items = append(items, line)
		}
		list = lipgloss.JoinVertical(lipgloss.Left, items...)
	}

	count := m.styles.Help.Render(fmt.Sprintf("Item %d of %d", m.cursor+1, len(m.items)))
	help := m.styles.Help.Render("j/k: navigate • x: select • r/l/a/d: action • 1/2/3: priority • enter: edit • q: quit")

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

func actionIcon(action string) string {
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

func keyMatches(msg tea.KeyMsg, target key.Binding) bool {
	for _, k := range target.Keys() {
		if msg.String() == k {
			return true
		}
	}
	return false
}
