package ui

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mcao2/readwise-triage/internal/config"
	"github.com/mcao2/readwise-triage/internal/readwise"
	"github.com/mcao2/readwise-triage/internal/triage"
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

	inboxLookback int
	feedLookback  int
	fetchLocation string
	editingDays   bool
	daysInput     string
	editingTags   bool
	tagsInput     string
	tagsCursor    int
}

type Item struct {
	ID           string
	Title        string
	Action       string
	Priority     string
	URL          string
	Summary      string
	Category     string
	Source       string
	WordCount    int
	ReadingTime  string
	Tags         []string // LLM-suggested tags
	OriginalTags []string // tags fetched from Readwise (preserved on update)
}

func NewModel() *Model {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{InboxDaysAgo: 7}
	}

	triageStore, err := config.LoadTriageStore()
	if err != nil {
		triageStore = nil // will be nil-checked by callers
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
		inboxLookback: cfg.InboxDaysAgo,
		feedLookback:  cfg.FeedDaysAgo,
		fetchLocation: "new",
	}

	// Restore last-used location from config
	if cfg.Location == "feed" {
		m.fetchLocation = "feed"
	}
	m.listView = NewListView(80, 24)
	m.listView.UpdateTableStyles(Themes[themeName])
	return m
}

func (m *Model) activeLookback() int {
	if m.fetchLocation == "feed" {
		return m.feedLookback
	}
	return m.inboxLookback
}

func (m *Model) activeLookbackPtr() *int {
	if m.fetchLocation == "feed" {
		return &m.feedLookback
	}
	return &m.inboxLookback
}

func (m *Model) saveLookback() {
	if m.cfg != nil {
		if m.fetchLocation == "feed" {
			m.cfg.FeedDaysAgo = m.feedLookback
		} else {
			m.cfg.InboxDaysAgo = m.inboxLookback
		}
		_ = m.cfg.Save()
	}
}

func (m *Model) saveLocation() {
	if m.cfg != nil {
		m.cfg.Location = m.fetchLocation
		_ = m.cfg.Save()
	}
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
		locationLabel := "inbox"
		if m.fetchLocation == "feed" {
			locationLabel = "feed"
		}
		m.statusMessage = fmt.Sprintf("Loaded %d %s items from the last %d days", len(m.items), locationLabel, m.activeLookback())
		m.state = StateReviewing

	case UpdateFinishedMsg:
		m.statusMessage = fmt.Sprintf("Successfully updated %d items (%d failed)", msg.Success, msg.Failed)
		m.state = StateDone

	case ErrorMsg:
		m.statusMessage = msg.Error.Error()
		m.state = StateConfig

	case TriageFinishedMsg:
		if msg.Err != nil {
			m.statusMessage = fmt.Sprintf("LLM triage failed: %v", msg.Err)
			m.messageType = "error"
			m.state = StateMessage
			return m, nil
		}
		applied := m.applyTriageResults(msg.Results)
		m.statusMessage = fmt.Sprintf("LLM auto-triaged %d items", applied)
		m.messageType = "success"
		m.state = StateMessage
	}

	return m, nil
}

func (m *Model) View() string {
	var content string
	centered := true

	switch m.state {
	case StateConfig:
		content = m.configView()
	case StateFetching:
		content = m.fetchingView()
	case StateTriaging:
		content = m.triagingView()
	case StateReviewing:
		content = m.reviewingView()
		centered = false
	case StateConfirming:
		content = m.confirmingView()
	case StateUpdating:
		content = m.updatingView()
	case StateDone:
		content = m.doneView()
	case StateMessage:
		content = m.messageView()
	default:
		return "Unknown state"
	}

	if centered && m.width > 0 && m.height > 0 {
		content = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	return content
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
	// Days input editing mode
	if m.editingDays {
		switch msg.Type {
		case tea.KeyEnter:
			if days, err := strconv.Atoi(m.daysInput); err == nil && days >= 1 {
				*m.activeLookbackPtr() = days
				m.saveLookback()
			}
			m.editingDays = false
			m.daysInput = ""
		case tea.KeyEsc:
			m.editingDays = false
			m.daysInput = ""
		case tea.KeyBackspace:
			if len(m.daysInput) > 0 {
				m.daysInput = m.daysInput[:len(m.daysInput)-1]
			}
		default:
			if len(msg.String()) == 1 && msg.String()[0] >= '0' && msg.String()[0] <= '9' {
				m.daysInput += msg.String()
			}
		}
		return m, nil
	}

	switch {
	case keyMatches(msg, m.keys.Enter):
		return m, m.startFetching()
	case keyMatches(msg, m.keys.CycleTheme):
		m.cycleTheme()
	case keyMatches(msg, m.keys.Left), keyMatches(msg, m.keys.Right):
		if m.fetchLocation == "new" {
			m.fetchLocation = "feed"
		} else {
			m.fetchLocation = "new"
		}
		m.saveLocation()
	case keyMatches(msg, m.keys.Up):
		*m.activeLookbackPtr() += 7
		m.saveLookback()
	case keyMatches(msg, m.keys.Down):
		if m.activeLookback() > 1 {
			*m.activeLookbackPtr() -= 7
			if m.activeLookback() < 1 {
				*m.activeLookbackPtr() = 1
			}
			m.saveLookback()
		}
	default:
		// Pressing a digit starts days editing mode
		if len(msg.String()) == 1 && msg.String()[0] >= '0' && msg.String()[0] <= '9' {
			m.editingDays = true
			m.daysInput = msg.String()
			return m, nil
		}
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
			DaysAgo:  m.activeLookback(),
			Location: m.fetchLocation,
		}
		items, err := client.GetInboxItems(opts)
		if err != nil {
			return ErrorMsg{Error: err}
		}

		uiItems := make([]Item, len(items))
		for i, item := range items {
			uiItems[i] = Item{
				ID:           item.ID,
				Title:        item.Title,
				Action:       "",
				Priority:     "",
				URL:          item.URL,
				Summary:      item.Summary,
				Category:     item.Category,
				Source:       item.Source,
				WordCount:    item.WordCount,
				ReadingTime:  item.ReadingTime,
				OriginalTags: []string(item.Tags),
			}
		}

		return ItemsLoadedMsg{Items: uiItems}
	}
}

// TriageFinishedMsg is sent when LLM auto-triage completes
type TriageFinishedMsg struct {
	Results []triage.Result
	Err     error
}

func (m *Model) startTriaging() tea.Cmd {
	m.state = StateTriaging

	return func() tea.Msg {
		if m.cfg == nil {
			return TriageFinishedMsg{Err: fmt.Errorf("configuration not loaded")}
		}

		llmCfg := m.cfg.GetLLMConfig()
		if llmCfg.Provider == "" && llmCfg.APIKey == "" {
			return TriageFinishedMsg{Err: fmt.Errorf("LLM not configured. Set llm.provider and llm.api_key in config.yaml or via LLM_API_KEY env var")}
		}

		client, err := triage.NewLLMClient(
			llmCfg.Provider,
			llmCfg.APIKey,
			triage.WithLLMBaseURL(llmCfg.BaseURL),
			triage.WithLLMModel(llmCfg.Model),
			triage.WithLLMAPIFormat(llmCfg.APIFormat),
		)
		if err != nil {
			return TriageFinishedMsg{Err: fmt.Errorf("failed to create LLM client: %w", err)}
		}

		// Build the items JSON (same logic as export)
		itemsJSON, err := m.buildTriageItemsJSON()
		if err != nil {
			return TriageFinishedMsg{Err: err}
		}

		results, err := client.TriageItems(itemsJSON)
		return TriageFinishedMsg{Results: results, Err: err}
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
				if m.fetchLocation == "feed" {
					update.Location = "new"
				}
			case "later":
				update.Location = "later"
			case "archive", "delete":
				update.Location = "archive"
			case "needs_review":
				if m.fetchLocation == "feed" {
					update.Location = "new"
				}
			}

			// Start with original Readwise tags to preserve them
			update.Tags = append(update.Tags, item.OriginalTags...)

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
	// Tag editing mode intercept
	if m.editingTags {
		runes := []rune(m.tagsInput)
		// Use msg.String() for word-jump bindings so both CSI sequences
		// (alt+left/alt+right) and ESC+letter sequences (alt+b/alt+f)
		// are handled — macOS terminals commonly send the latter.
		switch s := msg.String(); {
		case msg.Type == tea.KeyEnter:
			tags := parseTags(m.tagsInput)
			if m.batchMode {
				m.applyBatchTags(tags)
			} else if item := m.listView.GetItem(m.listView.Cursor()); item != nil {
				item.Tags = tags
				m.saveTriage(item.ID, item.Action, item.Priority, item.Tags)
				m.listView.SetItems(m.items)
			}
			m.editingTags = false
			m.tagsInput = ""
			m.tagsCursor = 0
		case msg.Type == tea.KeyEsc:
			m.editingTags = false
			m.tagsInput = ""
			m.tagsCursor = 0
		case msg.Type == tea.KeyBackspace && !msg.Alt:
			if m.tagsCursor > 0 {
				runes = append(runes[:m.tagsCursor-1], runes[m.tagsCursor:]...)
				m.tagsCursor--
				m.tagsInput = string(runes)
			}
		case s == "alt+backspace":
			// Option+Delete: delete previous word
			newPos := prevWordBoundary(runes, m.tagsCursor)
			runes = append(runes[:newPos], runes[m.tagsCursor:]...)
			m.tagsCursor = newPos
			m.tagsInput = string(runes)
		case s == "alt+left" || s == "alt+b":
			m.tagsCursor = prevWordBoundary(runes, m.tagsCursor)
		case s == "alt+right" || s == "alt+f":
			m.tagsCursor = nextWordBoundary(runes, m.tagsCursor)
		case msg.Type == tea.KeyLeft:
			if m.tagsCursor > 0 {
				m.tagsCursor--
			}
		case msg.Type == tea.KeyRight:
			if m.tagsCursor < len(runes) {
				m.tagsCursor++
			}
		default:
			if len(s) == 1 && s[0] >= 32 {
				r := []rune(s)[0]
				runes = append(runes[:m.tagsCursor], append([]rune{r}, runes[m.tagsCursor:]...)...)
				m.tagsCursor++
				m.tagsInput = string(runes)
			}
		}
		return m, nil
	}

	switch {
	case keyMatches(msg, m.keys.Enter):
		// Enter tag editing mode
		m.editingTags = true
		if m.batchMode {
			m.tagsInput = ""
			m.tagsCursor = 0
		} else if item := m.listView.GetItem(m.listView.Cursor()); item != nil {
			m.tagsInput = strings.Join(item.Tags, ", ")
			m.tagsCursor = len([]rune(m.tagsInput))
		}
		return m, nil
	case keyMatches(msg, m.keys.Up):
		// Use SetCursor directly to avoid the table's broken YOffset logic in MoveUp/MoveDown
		m.listView.MoveCursor(-1)
		m.cursor = m.listView.Cursor()
		return m, nil
	case keyMatches(msg, m.keys.Down):
		m.listView.MoveCursor(1)
		m.cursor = m.listView.Cursor()
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
			m.statusMessage = "Items exported to clipboard! Paste to your LLM."
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
		*m.activeLookbackPtr() += 7
		m.saveLookback()
		return m, m.startFetching()
	case keyMatches(msg, m.keys.Refresh):
		return m, m.startFetching()
	case keyMatches(msg, m.keys.AutoTriage):
		return m, m.startTriaging()
	case keyMatches(msg, m.keys.Back):
		m.state = StateConfig
		return m, nil
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
			m.saveTriage(m.items[idx].ID, m.items[idx].Action, m.items[idx].Priority, m.items[idx].Tags)
		}
	}
	m.listView.SetItems(m.items)
}

func (m *Model) applyBatchPriority(priority string) {
	selected := m.listView.GetSelected()
	for _, idx := range selected {
		if idx >= 0 && idx < len(m.items) {
			m.items[idx].Priority = priority
			m.saveTriage(m.items[idx].ID, m.items[idx].Action, m.items[idx].Priority, m.items[idx].Tags)
		}
	}
	m.listView.SetItems(m.items)
}

func (m *Model) applyBatchTags(tags []string) {
	selected := m.listView.GetSelected()
	for _, idx := range selected {
		if idx >= 0 && idx < len(m.items) {
			m.items[idx].Tags = tags
			m.saveTriage(m.items[idx].ID, m.items[idx].Action, m.items[idx].Priority, m.items[idx].Tags)
		}
	}
	m.listView.SetItems(m.items)
}

func parseTags(input string) []string {
	parts := strings.Split(input, ",")
	var tags []string
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

// prevWordBoundary returns the cursor position at the start of the previous word.
func prevWordBoundary(runes []rune, pos int) int {
	if pos <= 0 {
		return 0
	}
	i := pos - 1
	// Skip trailing whitespace/punctuation
	for i > 0 && !unicode.IsLetter(runes[i]) && !unicode.IsDigit(runes[i]) {
		i--
	}
	// Skip the word
	for i > 0 && (unicode.IsLetter(runes[i-1]) || unicode.IsDigit(runes[i-1])) {
		i--
	}
	return i
}

// nextWordBoundary returns the cursor position at the end of the next word.
func nextWordBoundary(runes []rune, pos int) int {
	n := len(runes)
	if pos >= n {
		return n
	}
	i := pos
	// Skip current whitespace/punctuation
	for i < n && !unicode.IsLetter(runes[i]) && !unicode.IsDigit(runes[i]) {
		i++
	}
	// Skip the word
	for i < n && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i])) {
		i++
	}
	return i
}

func (m *Model) setItemAction(item *Item, action string) {
	item.Action = action
	m.saveTriage(item.ID, item.Action, item.Priority, item.Tags)
	m.listView.SetItems(m.items)
}

func (m *Model) setItemPriority(item *Item, priority string) {
	item.Priority = priority
	m.saveTriage(item.ID, item.Action, item.Priority, item.Tags)
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
	m.state = StateReviewing
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
			m.items[i].Tags = entry.Tags
		}
	}
}

func (m *Model) saveTriage(id, action, priority string, tags []string) {
	if m.triageStore == nil {
		return
	}
	m.triageStore.SetItem(id, action, priority, "manual", tags, nil)
}

func (m *Model) saveLLMTriage(id, action, priority string, tags []string, report *triage.Result) {
	if m.triageStore == nil {
		return
	}
	m.triageStore.SetItem(id, action, priority, "llm", tags, report)
}

func (m *Model) configView() string {
	// Styled title
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(m.styles.theme.Primary)).
		Render("  Readwise Triage")

	// Theme indicator
	themeName := m.cfg.Theme
	if themeName == "" {
		themeName = "default"
	}
	themeLine := fmt.Sprintf("  🎨  %s", m.styles.Normal.Render("Theme: "+themeName))

	// Location indicator
	locationLabel := "Inbox"
	if m.fetchLocation == "feed" {
		locationLabel = "Feed"
	}
	locationLine := fmt.Sprintf("  📂  %s", m.styles.Normal.Render("Location: "+locationLabel))

	// Days lookback
	var daysLine string
	if m.editingDays {
		daysLine = fmt.Sprintf("  📅  %s", m.styles.Normal.Render("Days: "+m.daysInput+"▌"))
	} else {
		daysLine = fmt.Sprintf("  📅  %s", m.styles.Normal.Render(fmt.Sprintf("Fetch last %d days", m.activeLookback())))
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		"",
		title,
		"",
		themeLine,
		locationLine,
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
		{"h/l", "location"},
		{"j/k", "days ±7"},
		{"0-9", "type days"},
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

	fetchTitle := "Fetching Inbox Items"
	if m.fetchLocation == "feed" {
		fetchTitle = "Fetching Feed Items"
	}

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
	locationTag := "[Inbox]"
	if m.fetchLocation == "feed" {
		locationTag = "[Feed]"
	}
	headerLeft := m.styles.HelpKey.Render("Readwise Triage " + locationTag)
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
		statusLine = m.styles.Help.Render("  " + m.statusMessage)
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

	help := m.renderHelpLine([]helpEntry{{"any key", "back to review"}})
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
		{"enter", "tags"},
		{"e", "export"},
		{"i", "import"},
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
			{"d", "delete"},
			{"n", "needs review"},
		}},
		{"Priority", []helpEntry{
			{"1", "high"},
			{"2", "medium"},
			{"3", "low"},
		}},
		{"Operations", []helpEntry{
			{"enter", "edit tags"},
			{"e", "export to clipboard"},
			{"i", "import from clipboard"},
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

// buildTriageItemsJSON builds the JSON payload for LLM triage.
// Selection-aware: uses selected items if any, otherwise untriaged items.
func (m *Model) buildTriageItemsJSON() (string, error) {
	type exportItem struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		URL         string `json:"url"`
		Summary     string `json:"summary"`
		Category    string `json:"category"`
		Source      string `json:"source"`
		WordCount   int    `json:"word_count"`
		ReadingTime string `json:"reading_time"`
	}

	var items []exportItem
	selectedIndices := m.listView.GetSelected()
	useSelection := len(selectedIndices) > 0

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
		} else if item.Action != "" {
			// Skip already-triaged items when no selection
			continue
		}

		items = append(items, exportItem{
			ID:          item.ID,
			Title:       item.Title,
			URL:         item.URL,
			Summary:     item.Summary,
			Category:    item.Category,
			Source:      item.Source,
			WordCount:   item.WordCount,
			ReadingTime: item.ReadingTime,
		})
	}

	if len(items) == 0 {
		return "", fmt.Errorf("no items to triage (all items already triaged)")
	}

	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal items: %w", err)
	}

	return string(data), nil
}

// applyTriageResults applies LLM triage results to the current items.
// Returns the number of items successfully applied.
func (m *Model) applyTriageResults(results []triage.Result) int {
	itemMap := make(map[string]*Item)
	for i := range m.items {
		itemMap[m.items[i].ID] = &m.items[i]
	}

	applied := 0
	for _, result := range results {
		item, ok := itemMap[result.ID]
		if !ok {
			continue
		}

		if result.TriageDecision.Action == "" {
			continue
		}

		item.Action = result.TriageDecision.Action
		item.Priority = result.TriageDecision.Priority

		// Apply suggested tags, filtering out action-name duplicates
		if len(result.MetadataEnhancement.SuggestedTags) > 0 {
			var filtered []string
			for _, tag := range result.MetadataEnhancement.SuggestedTags {
				lower := strings.ToLower(strings.TrimSpace(tag))
				if !validActions[lower] {
					filtered = append(filtered, tag)
				}
			}
			item.Tags = filtered
		}

		// Save to triage store with full report
		m.saveLLMTriage(item.ID, item.Action, item.Priority, item.Tags, &result)
		applied++
	}

	m.listView.SetItems(m.items)
	return applied
}
