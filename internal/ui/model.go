package ui

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
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
	StateFetching State = iota
	StateReviewing
	StateConfirming
	StateUpdating
	StateDone
	StateMessage
)

func (s State) String() string {
	switch s {
	case StateFetching:
		return "Fetching"
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

	showHelp bool

	items  []Item
	cursor int

	listView ListView
	spinner  spinner.Model
	progress progress.Model

	updateProgress float64
	statusMessage  string
	messageType    string
	batchMode      bool

	triaging    bool
	triagingIDs map[string]bool

	cfg         *config.Config
	triageStore *config.TriageStore

	inboxLookback int
	editingTags   bool
	tagsInput     string
	tagsCursor    int
}

type Item struct {
	ID           string
	Title        string
	Action       string
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

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(DefaultTheme.Primary))

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithoutPercentage(),
	)

	m := &Model{
		state:         StateFetching,
		styles:        NewStyles(DefaultTheme),
		keys:          DefaultKeyMap(),
		items:         []Item{},
		cursor:        0,
		spinner:       s,
		progress:      p,
		cfg:           cfg,
		triageStore:   triageStore,
		inboxLookback: cfg.InboxDaysAgo,
	}

	m.listView = NewListView(80, 24)
	m.listView.UpdateTableStyles(DefaultTheme)
	return m
}

func (m *Model) activeLookback() int {
	return m.inboxLookback
}

func (m *Model) activeLookbackPtr() *int {
	return &m.inboxLookback
}

func (m *Model) saveLookback() {
	if m.cfg != nil {
		m.cfg.InboxDaysAgo = m.inboxLookback
		_ = m.cfg.Save()
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.startFetching())
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
		m.statusMessage = fmt.Sprintf("Loaded %d items from the last %d days", len(m.items), m.activeLookback())
		m.state = StateReviewing

	case UpdateFinishedMsg:
		m.statusMessage = fmt.Sprintf("Successfully updated %d items (%d failed)", msg.Success, msg.Failed)
		m.state = StateDone

	case ErrorMsg:
		m.statusMessage = msg.Error.Error()
		m.state = StateMessage

	case TriageProgressMsg:
		m.statusMessage = fmt.Sprintf("Triaging batch %d/%d...", msg.Batch, msg.Total)
		m.applyTriageResults(msg.Results)
		m.listView.SetItems(m.items)
		m.listView.SetTriagingIDs(m.triagingIDs)
		return m, m.waitForTriageProgress(msg.Channel, msg.Results)

	case TriageFinishedMsg:
		m.triaging = false
		m.triagingIDs = nil
		m.listView.SetTriagingIDs(nil)
		m.listView.SetItems(m.items)
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
	case StateFetching:
		content = m.fetchingView()
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
			Location: "new",
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

// TriageBatchProgress is sent on the progress channel after each batch.
type TriageBatchProgress struct {
	Batch   int
	Total   int
	Results []triage.Result
	Err     error
}

// TriageProgressMsg is handled by Update() to show per-batch progress.
type TriageProgressMsg struct {
	Batch   int
	Total   int
	Results []triage.Result
	Channel chan TriageBatchProgress
}

func (m *Model) startTriaging() tea.Cmd {
	m.triaging = true

	// Get items synchronously (fast — just filters in-memory)
	items, err := m.getTriageItems()
	if err != nil {
		m.triaging = false
		m.statusMessage = fmt.Sprintf("Triage failed: %v", err)
		m.state = StateMessage
		return nil
	}

	numBatches := (len(items) + triageBatchSize - 1) / triageBatchSize
	m.statusMessage = fmt.Sprintf("Triaging batch 1/%d...", numBatches)

	// Mark items as being triaged
	m.triagingIDs = make(map[string]bool)
	for _, item := range items {
		m.triagingIDs[item.ID] = true
	}
	m.listView.SetItems(m.items)
	m.listView.SetTriagingIDs(m.triagingIDs)

	return func() tea.Msg {
		if m.cfg == nil {
			return TriageFinishedMsg{Err: fmt.Errorf("configuration not loaded")}
		}

		llmCfg := m.cfg.GetLLMConfig()
		if llmCfg.APIKey == "" && llmCfg.BaseURL == "" {
			return TriageFinishedMsg{Err: fmt.Errorf("LLM not configured. Set llm.api_key and llm.base_url in config.yaml or via LLM_API_KEY and LLM_BASE_URL env vars")}
		}

		client, err := triage.NewLLMClient(
			llmCfg.APIKey,
			triage.WithLLMBaseURL(llmCfg.BaseURL),
			triage.WithLLMModel(llmCfg.Model),
			triage.WithLLMMaxTokens(llmCfg.MaxTokens),
		)
		if err != nil {
			return TriageFinishedMsg{Err: fmt.Errorf("failed to create LLM client: %w", err)}
		}

		progressChan := make(chan TriageBatchProgress)

		go func() {
			for batchStart := 0; batchStart < len(items); batchStart += triageBatchSize {
				batchNum := batchStart/triageBatchSize + 1
				batchEnd := batchStart + triageBatchSize
				if batchEnd > len(items) {
					batchEnd = len(items)
				}

				batchBytes, err := json.MarshalIndent(items[batchStart:batchEnd], "", "  ")
				if err != nil {
					progressChan <- TriageBatchProgress{Batch: batchNum, Total: numBatches, Err: fmt.Errorf("failed to marshal batch: %w", err)}
					break
				}

				results, err := client.TriageItems(string(batchBytes))
				if err != nil {
					progressChan <- TriageBatchProgress{Batch: batchNum, Total: numBatches, Err: fmt.Errorf("batch %d/%d failed: %w", batchNum, numBatches, err)}
					break
				}

				progressChan <- TriageBatchProgress{Batch: batchNum, Total: numBatches, Results: results}

				if batchEnd < len(items) {
					time.Sleep(1 * time.Second)
				}
			}
			close(progressChan)
		}()

		return m.waitForTriageProgress(progressChan, nil)
	}
}

func (m *Model) waitForTriageProgress(ch chan TriageBatchProgress, allResults []triage.Result) tea.Cmd {
	return func() tea.Msg {
		progress, ok := <-ch
		if !ok {
			return TriageFinishedMsg{Results: allResults}
		}
		if progress.Err != nil {
			return TriageFinishedMsg{Err: progress.Err}
		}
		return TriageProgressMsg{
			Batch:   progress.Batch,
			Total:   progress.Total,
			Results: append(allResults, progress.Results...),
			Channel: ch,
		}
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
				// stays in inbox
			case "later":
				update.Location = "later"
			case "archive":
				update.Location = "archive"
			}

			// Start with original Readwise tags to preserve them
			update.Tags = append(update.Tags, item.OriginalTags...)

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
	// During triage: allow only navigation, open URL, help, quit
	if m.triaging {
		switch {
		case keyMatches(msg, m.keys.Quit):
			return m, tea.Quit
		case keyMatches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil
		case keyMatches(msg, m.keys.Up):
			m.listView.SetCursor(m.listView.Cursor() - 1)
			return m, nil
		case keyMatches(msg, m.keys.Down):
			m.listView.SetCursor(m.listView.Cursor() + 1)
			return m, nil
		case keyMatches(msg, m.keys.Open):
			if item := m.listView.GetItem(m.listView.Cursor()); item != nil && item.URL != "" {
				openURL(item.URL)
			}
			return m, nil
		default:
			return m, nil
		}
	}

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
				m.saveTriage(item.ID, item.Action, item.Tags)
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
		// SetCursor directly for clean cursor management
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
		return m, m.startFetching()
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
			m.applyBatchAction("archive")
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
			m.setItemAction(item, "archive")
		}
	}

	return m, nil
}

func (m *Model) applyBatchAction(action string) {
	selected := m.listView.GetSelected()
	for _, idx := range selected {
		if idx >= 0 && idx < len(m.items) {
			m.items[idx].Action = action
			m.saveTriage(m.items[idx].ID, m.items[idx].Action, m.items[idx].Tags)
		}
	}
	m.listView.SetItems(m.items)
}

func (m *Model) applyBatchTags(tags []string) {
	selected := m.listView.GetSelected()
	for _, idx := range selected {
		if idx >= 0 && idx < len(m.items) {
			m.items[idx].Tags = tags
			m.saveTriage(m.items[idx].ID, m.items[idx].Action, m.items[idx].Tags)
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
	m.saveTriage(item.ID, item.Action, item.Tags)
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
	return m, m.startFetching()
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
			m.items[i].Tags = entry.Tags
		}
	}
}

func (m *Model) saveTriage(id, action string, tags []string) {
	if m.triageStore == nil {
		return
	}
	m.triageStore.SetItem(id, action, "manual", tags, nil)
}

func (m *Model) saveLLMTriage(id, action string, tags []string, report *triage.Result) {
	if m.triageStore == nil {
		return
	}
	m.triageStore.SetItem(id, action, "llm", tags, report)
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

const triageBatchSize = 20

// triageItem is the JSON payload sent to the LLM for each item.
type triageItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Summary     string `json:"summary"`
	Category    string `json:"category"`
	Source      string `json:"source"`
	WordCount   int    `json:"word_count"`
	ReadingTime string `json:"reading_time"`
}

// getTriageItems returns the items to send to the LLM for triage.
// Selection-aware: uses selected items if any, otherwise untriaged items.
func (m *Model) getTriageItems() ([]triageItem, error) {

	var items []triageItem
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

		items = append(items, triageItem{
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
		return nil, fmt.Errorf("no items to triage (all items already triaged)")
	}

	return items, nil
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

		// Apply suggested tags, filtering out action-name duplicates
		if len(result.MetadataEnhancement.SuggestedTags) > 0 {
			var filtered []string
			for _, tag := range result.MetadataEnhancement.SuggestedTags {
				lower := strings.ToLower(strings.TrimSpace(tag))
				if !triage.ValidActions[lower] {
					filtered = append(filtered, tag)
				}
			}
			item.Tags = filtered
		}

		// Save to triage store with full report
		m.saveLLMTriage(item.ID, item.Action, item.Tags, &result)
		applied++
	}

	m.listView.SetItems(m.items)
	return applied
}
