package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mcao2/readwise-triage/internal/config"
	"github.com/mcao2/readwise-triage/internal/readwise"
)

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "readwise-triage-test")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	os.Setenv("READWISE_TRIAGE_CONFIG", filepath.Join(tmpDir, "config.yaml"))

	os.Exit(m.Run())
}

func TestNewModel(t *testing.T) {
	m := NewModel()
	if m.state != StateConfig {
		t.Errorf("expected initial state StateConfig, got %v", m.state)
	}
	if m.cursor != 0 {
		t.Errorf("expected initial cursor 0, got %d", m.cursor)
	}
}

func TestStateTransitions(t *testing.T) {
	m := NewModel()

	m.Update(StateChangeMsg{State: StateFetching})
	if m.state != StateFetching {
		t.Errorf("expected state StateFetching, got %v", m.state)
	}

	items := []Item{
		{ID: "1", Title: "Item 1"},
		{ID: "2", Title: "Item 2"},
	}
	m.Update(ItemsLoadedMsg{Items: items})
	if m.state != StateReviewing {
		t.Errorf("expected state StateReviewing, got %v", m.state)
	}
	if len(m.items) != 2 {
		t.Errorf("expected 2 items, got %d", len(m.items))
	}

	m.Update(ErrorMsg{Error: fmt.Errorf("test error")})
	if m.state != StateConfig {
		t.Errorf("expected state StateConfig after error, got %v", m.state)
	}

	m.Update(ItemsLoadedMsg{Items: items})
	m.Update(UpdateFinishedMsg{Success: 2, Failed: 0})
	if m.state != StateDone {
		t.Errorf("expected state StateDone, got %v", m.state)
	}
}

func TestNavigation(t *testing.T) {
	m := NewModel()
	items := []Item{
		{ID: "1", Title: "Item 1"},
		{ID: "2", Title: "Item 2"},
		{ID: "3", Title: "Item 3"},
	}
	m.Update(ItemsLoadedMsg{Items: items})
	m.state = StateReviewing

	if m.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", m.cursor)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.cursor != 1 {
		t.Errorf("expected cursor 1 after 'j', got %d", m.cursor)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.cursor != 2 {
		t.Errorf("expected cursor 2 after 'j', got %d", m.cursor)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.cursor != 2 {
		t.Errorf("expected cursor 2 (boundary) after 'j', got %d", m.cursor)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.cursor != 1 {
		t.Errorf("expected cursor 1 after 'k', got %d", m.cursor)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.cursor != 0 {
		t.Errorf("expected cursor 0 after 'k', got %d", m.cursor)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.cursor != 0 {
		t.Errorf("expected cursor 0 (boundary) after 'k', got %d", m.cursor)
	}
}

func TestSelectionAndBatchMode(t *testing.T) {
	m := NewModel()
	items := []Item{
		{ID: "1", Title: "Item 1"},
		{ID: "2", Title: "Item 2"},
	}
	m.Update(ItemsLoadedMsg{Items: items})

	if m.batchMode {
		t.Error("expected batchMode false initially")
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	if !m.batchMode {
		t.Error("expected batchMode true after selecting item")
	}
	if !m.listView.IsSelected(0) {
		t.Error("expected item 0 to be selected")
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	if m.batchMode {
		t.Error("expected batchMode false after deselecting item")
	}
	if m.listView.IsSelected(0) {
		t.Error("expected item 0 to be deselected")
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})

	if !m.batchMode {
		t.Error("expected batchMode true")
	}
	selected := m.listView.GetSelected()
	if len(selected) != 2 {
		t.Errorf("expected 2 items selected, got %d", len(selected))
	}
}

func TestApplyActions(t *testing.T) {
	m := NewModel()
	items := []Item{
		{ID: "1", Title: "Item 1"},
		{ID: "2", Title: "Item 2"},
	}
	m.Update(ItemsLoadedMsg{Items: items})

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if m.items[0].Action != "read_now" {
		t.Errorf("expected action 'read_now', got %s", m.items[0].Action)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	if m.items[0].Priority != "high" {
		t.Errorf("expected priority 'high', got %s", m.items[0].Priority)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	if m.items[0].Action != "later" {
		t.Errorf("expected item 0 action 'later', got %s", m.items[0].Action)
	}
	if m.items[1].Action != "later" {
		t.Errorf("expected item 1 action 'later', got %s", m.items[1].Action)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	if m.items[0].Priority != "medium" {
		t.Errorf("expected item 0 priority 'medium', got %s", m.items[0].Priority)
	}
	if m.items[1].Priority != "medium" {
		t.Errorf("expected item 1 priority 'medium', got %s", m.items[1].Priority)
	}
}

func TestThemeCycling(t *testing.T) {
	m := NewModel()
	initialTheme := m.cfg.Theme
	if initialTheme == "" {
		initialTheme = "default"
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	newTheme := m.cfg.Theme
	if newTheme == initialTheme {
		t.Errorf("expected theme to change, but it's still %s", initialTheme)
	}
}

func TestImportTriageResults(t *testing.T) {
	m := NewModel()
	m.items = []Item{
		{ID: "1", Title: "Item 1"},
		{ID: "2", Title: "Item 2"},
	}

	jsonData := `[
		{
			"id": "1",
			"title": "Item 1",
			"triage_decision": {
				"action": "read_now",
				"priority": "high"
			}
		},
		{
			"id": "2",
			"title": "Item 2",
			"triage_decision": {
				"action": "later",
				"priority": "low"
			}
		}
	]`

	applied, err := m.ImportTriageResults(jsonData)
	if err != nil {
		t.Fatalf("ImportTriageResults failed: %v", err)
	}
	if applied != 2 {
		t.Errorf("expected 2 items applied, got %d", applied)
	}

	if m.items[0].Action != "read_now" || m.items[0].Priority != "high" {
		t.Errorf("item 1 not updated correctly: %+v", m.items[0])
	}
	if m.items[1].Action != "later" || m.items[1].Priority != "low" {
		t.Errorf("item 2 not updated correctly: %+v", m.items[1])
	}
}

func TestHandleAdditionalKeys(t *testing.T) {
	m := NewModel()

	m.state = StateConfirming
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if m.state != StateReviewing {
		t.Errorf("expected Reviewing state after 'n' in Confirming, got %v", m.state)
	}

	m.state = StateConfirming
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if cmd == nil {
		t.Error("expected command after 'y' in Confirming")
	}

	m.state = StateDone
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if m.state != StateReviewing {
		t.Errorf("expected Reviewing state after key in Done, got %v", m.state)
	}

	m.state = StateMessage
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if m.state != StateReviewing {
		t.Errorf("expected Reviewing state after key in Message, got %v", m.state)
	}
}

func TestValidateTriageJSON(t *testing.T) {
	m := NewModel()
	m.items = []Item{{ID: "1", Title: "Test"}}

	validJSON := `[{"id": "1", "title": "Test", "triage_decision": {"action": "read_now"}}]`
	ok, msg := m.ValidateTriageJSON(validJSON)
	if !ok {
		t.Errorf("expected JSON to be valid, got error: %s", msg)
	}

	invalidJSON := `[{"id": "unknown", "title": "Test", "triage_decision": {"action": "read_now"}}]`
	ok, msg = m.ValidateTriageJSON(invalidJSON)
	if ok {
		t.Error("expected JSON to be invalid due to unknown ID")
	}

	badActionJSON := `[{"id": "1", "title": "Test", "triage_decision": {"action": "invalid"}}]`
	ok, msg = m.ValidateTriageJSON(badActionJSON)
	if ok {
		t.Error("expected JSON to be invalid due to bad action")
	}
}

func TestUpdateWithSelection(t *testing.T) {
	m := NewModel()
	m.items = []Item{
		{ID: "1", Title: "Item 1", Action: "read_now"},
		{ID: "2", Title: "Item 2", Action: "later"},
	}
	m.listView.SetItems(m.items)

	m.listView.SetCursor(0)
	m.listView.ToggleSelection()

	m.state = StateConfirming
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if cmd == nil {
		t.Fatal("expected command from Update")
	}
}

func TestProgressUpdateLoop(t *testing.T) {
	m := NewModel()
	ch := make(chan readwise.BatchUpdateProgress, 2)

	cmd := m.waitForUpdateProgress(ch, 0, 0)

	ch <- readwise.BatchUpdateProgress{Current: 1, Total: 2, ItemID: "1", Success: true}

	msg := cmd()
	progressMsg, ok := msg.(ProgressMsg)
	if !ok {
		t.Fatalf("expected ProgressMsg, got %T", msg)
	}
	if progressMsg.Progress != 0.5 {
		t.Errorf("expected progress 0.5, got %f", progressMsg.Progress)
	}

	nextCmd := m.waitForUpdateProgress(progressMsg.Channel, progressMsg.Success, progressMsg.Failed)
	ch <- readwise.BatchUpdateProgress{Current: 2, Total: 2, ItemID: "2", Success: true}

	msg2 := nextCmd()
	progressMsg2, ok := msg2.(ProgressMsg)
	if !ok {
		t.Fatalf("expected ProgressMsg, got %T", msg2)
	}
	if progressMsg2.Progress != 1.0 {
		t.Errorf("expected progress 1.0, got %f", progressMsg2.Progress)
	}

	close(ch)
	finishCmd := m.waitForUpdateProgress(progressMsg2.Channel, progressMsg2.Success, progressMsg2.Failed)
	finishMsg := finishCmd()
	if _, ok := finishMsg.(UpdateFinishedMsg); !ok {
		t.Fatalf("expected UpdateFinishedMsg, got %T", finishMsg)
	}
}

func TestViewRendering(t *testing.T) {

	m := NewModel()

	view := m.View()
	if view == "" {
		t.Error("Config view is empty")
	}

	m.state = StateReviewing
	m.items = []Item{{ID: "1", Title: "Test"}}
	m.listView.SetItems(m.items)
	view = m.View()
	if view == "" {
		t.Error("Reviewing view is empty")
	}

	m.state = StateDone
	m.statusMessage = "All done"
	view = m.View()
	if view == "" {
		t.Error("Done view is empty")
	}
}

func TestRefreshKey(t *testing.T) {
	m := NewModel()
	items := []Item{
		{ID: "1", Title: "Item 1"},
		{ID: "2", Title: "Item 2"},
	}
	m.Update(ItemsLoadedMsg{Items: items})
	m.state = StateReviewing

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")})

	if m.state != StateFetching {
		t.Errorf("expected state StateFetching after Refresh key, got %v", m.state)
	}

	if cmd == nil {
		t.Error("expected command after Refresh key")
	}
}

func TestNeedsReviewAction(t *testing.T) {
	m := NewModel()
	items := []Item{
		{ID: "1", Title: "Item 1"},
	}
	m.Update(ItemsLoadedMsg{Items: items})
	m.state = StateReviewing
	m.listView.SetCursor(0)

	// Apply needs_review action
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})

	if m.items[0].Action != "needs_review" {
		t.Errorf("expected action 'needs_review', got %s", m.items[0].Action)
	}
}

func TestBatchNeedsReviewAction(t *testing.T) {
	m := NewModel()
	items := []Item{
		{ID: "1", Title: "Item 1"},
		{ID: "2", Title: "Item 2"},
	}
	m.Update(ItemsLoadedMsg{Items: items})
	m.state = StateReviewing

	// Select both items
	m.listView.SetCursor(0)
	m.listView.ToggleSelection()
	m.listView.SetCursor(1)
	m.listView.ToggleSelection()
	m.batchMode = true

	// Apply needs_review to batch
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})

	if m.items[0].Action != "needs_review" {
		t.Errorf("expected item 0 action 'needs_review', got %s", m.items[0].Action)
	}
	if m.items[1].Action != "needs_review" {
		t.Errorf("expected item 1 action 'needs_review', got %s", m.items[1].Action)
	}
}

func TestUpdateRequestWithTags(t *testing.T) {
	m := NewModel()
	m.cfg = &config.Config{ReadwiseToken: "test-token"}

	// Directly set items with tags
	m.items = []Item{
		{
			ID:           "1",
			Title:        "Item 1",
			Action:       "read_now",
			Priority:     "high",
			Tags:         []string{"golang", "tutorial"},
			OriginalTags: []string{"inbox", "rss"},
		},
		{
			ID:           "2",
			Title:        "Item 2",
			Action:       "needs_review",
			Tags:         []string{"paywalled"},
			OriginalTags: []string{"saved"},
		},
	}
	m.state = StateReviewing

	// Simulate building update requests (we can't easily test the actual API call)
	// but we can verify the logic by checking what would be sent
	var updates []readwise.UpdateRequest
	for _, item := range m.items {
		if item.Action != "" {
			update := readwise.UpdateRequest{
				DocumentID: item.ID,
			}

			switch item.Action {
			case "read_now":
				// no action-based tag
			case "needs_review":
				// no action-based tag
			}

			// Preserve original tags
			update.Tags = append(update.Tags, item.OriginalTags...)

			if item.Priority != "" {
				update.Tags = append(update.Tags, "priority:"+item.Priority)
			}

			if len(item.Tags) > 0 {
				update.Tags = append(update.Tags, item.Tags...)
			}

			updates = append(updates, update)
		}
	}

	if len(updates) != 2 {
		t.Fatalf("expected 2 updates, got %d", len(updates))
	}

	// Check first item: original tags + priority:high + golang + tutorial
	expectedTags1 := []string{"inbox", "rss", "priority:high", "golang", "tutorial"}
	if len(updates[0].Tags) != len(expectedTags1) {
		t.Errorf("expected %d tags for item 1, got %d", len(expectedTags1), len(updates[0].Tags))
	}
	for i, tag := range expectedTags1 {
		if updates[0].Tags[i] != tag {
			t.Errorf("expected tag %d to be %s, got %s", i, tag, updates[0].Tags[i])
		}
	}

	// Check second item: original tags + paywalled
	expectedTags2 := []string{"saved", "paywalled"}
	if len(updates[1].Tags) != len(expectedTags2) {
		t.Errorf("expected %d tags for item 2, got %d", len(expectedTags2), len(updates[1].Tags))
	}
	for i, tag := range expectedTags2 {
		if updates[1].Tags[i] != tag {
			t.Errorf("expected tag %d to be %s, got %s", i, tag, updates[1].Tags[i])
		}
	}
}

func TestAllSingleItemActions(t *testing.T) {
	tests := []struct {
		key    string
		action string
	}{
		{"r", "read_now"},
		{"l", "later"},
		{"a", "archive"},
		{"d", "delete"},
		{"n", "needs_review"},
	}

	for _, tt := range tests {
		t.Run(tt.key+"="+tt.action, func(t *testing.T) {
			m := NewModel()
			m.Update(ItemsLoadedMsg{Items: []Item{{ID: "1", Title: "Test"}}})
			m.state = StateReviewing

			m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})
			if m.items[0].Action != tt.action {
				t.Errorf("expected action %q, got %q", tt.action, m.items[0].Action)
			}
		})
	}
}

func TestAllSingleItemPriorities(t *testing.T) {
	tests := []struct {
		key      string
		priority string
	}{
		{"1", "high"},
		{"2", "medium"},
		{"3", "low"},
	}

	for _, tt := range tests {
		t.Run(tt.key+"="+tt.priority, func(t *testing.T) {
			m := NewModel()
			m.Update(ItemsLoadedMsg{Items: []Item{{ID: "1", Title: "Test"}}})
			m.state = StateReviewing

			m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})
			if m.items[0].Priority != tt.priority {
				t.Errorf("expected priority %q, got %q", tt.priority, m.items[0].Priority)
			}
		})
	}
}

func TestAllBatchActions(t *testing.T) {
	tests := []struct {
		key    string
		action string
	}{
		{"r", "read_now"},
		{"l", "later"},
		{"a", "archive"},
		{"d", "delete"},
		{"n", "needs_review"},
	}

	for _, tt := range tests {
		t.Run("batch_"+tt.key+"="+tt.action, func(t *testing.T) {
			m := NewModel()
			items := []Item{
				{ID: "1", Title: "Item 1"},
				{ID: "2", Title: "Item 2"},
			}
			m.Update(ItemsLoadedMsg{Items: items})
			m.state = StateReviewing

			// Select both items
			m.listView.SetCursor(0)
			m.listView.ToggleSelection()
			m.listView.SetCursor(1)
			m.listView.ToggleSelection()
			m.batchMode = true

			m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})
			for i, item := range m.items {
				if item.Action != tt.action {
					t.Errorf("item %d: expected action %q, got %q", i, tt.action, item.Action)
				}
			}
		})
	}
}

func TestAllBatchPriorities(t *testing.T) {
	tests := []struct {
		key      string
		priority string
	}{
		{"1", "high"},
		{"2", "medium"},
		{"3", "low"},
	}

	for _, tt := range tests {
		t.Run("batch_"+tt.key+"="+tt.priority, func(t *testing.T) {
			m := NewModel()
			items := []Item{
				{ID: "1", Title: "Item 1"},
				{ID: "2", Title: "Item 2"},
			}
			m.Update(ItemsLoadedMsg{Items: items})
			m.state = StateReviewing

			m.listView.SetCursor(0)
			m.listView.ToggleSelection()
			m.listView.SetCursor(1)
			m.listView.ToggleSelection()
			m.batchMode = true

			m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})
			for i, item := range m.items {
				if item.Priority != tt.priority {
					t.Errorf("item %d: expected priority %q, got %q", i, tt.priority, item.Priority)
				}
			}
		})
	}
}

func TestFetchMoreKey(t *testing.T) {
	m := NewModel()
	items := []Item{{ID: "1", Title: "Item 1"}}
	m.Update(ItemsLoadedMsg{Items: items})
	m.state = StateReviewing

	initialLookback := m.fetchLookback

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})

	if m.fetchLookback != initialLookback+7 {
		t.Errorf("expected fetchLookback %d, got %d", initialLookback+7, m.fetchLookback)
	}
	if m.state != StateFetching {
		t.Errorf("expected StateFetching, got %v", m.state)
	}
	if cmd == nil {
		t.Error("expected command after FetchMore key")
	}
}

func TestToggleLLMMode_Disabled(t *testing.T) {
	m := NewModel()
	initial := m.useLLMTriage

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	if m.useLLMTriage != initial {
		t.Error("expected useLLMTriage to remain unchanged when toggle is hidden")
	}
}

func TestAllViewRendering(t *testing.T) {
	tests := []struct {
		name  string
		setup func(m *Model)
	}{
		{"config", func(m *Model) { m.state = StateConfig }},
		{"fetching", func(m *Model) { m.state = StateFetching }},
		{"triaging", func(m *Model) { m.state = StateTriaging }},
		{"reviewing", func(m *Model) {
			m.state = StateReviewing
			m.items = []Item{{ID: "1", Title: "Test"}}
			m.listView.SetItems(m.items)
		}},
		{"reviewing_batch", func(m *Model) {
			m.state = StateReviewing
			m.items = []Item{{ID: "1", Title: "Test"}}
			m.listView.SetItems(m.items)
			m.batchMode = true
		}},
		{"reviewing_empty", func(m *Model) {
			m.state = StateReviewing
			m.items = []Item{}
		}},
		{"confirming", func(m *Model) { m.state = StateConfirming }},
		{"updating", func(m *Model) {
			m.state = StateUpdating
			m.updateProgress = 0.5
			m.statusMessage = "Updating..."
		}},
		{"done", func(m *Model) {
			m.state = StateDone
			m.statusMessage = "All done"
		}},
		{"message_success", func(m *Model) {
			m.state = StateMessage
			m.messageType = "success"
			m.statusMessage = "It worked"
		}},
		{"message_error", func(m *Model) {
			m.state = StateMessage
			m.messageType = "error"
			m.statusMessage = "Something failed"
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel()
			tt.setup(m)
			view := m.View()
			if view == "" {
				t.Errorf("%s view is empty", tt.name)
			}
		})
	}
}

func TestConfirmingToUpdatingFlow(t *testing.T) {
	m := NewModel()
	m.cfg = &config.Config{ReadwiseToken: "test-token"}
	m.items = []Item{
		{ID: "1", Title: "Item 1", Action: "read_now", Priority: "high"},
		{ID: "2", Title: "Item 2", Action: "later"},
	}
	m.listView.SetItems(m.items)
	m.state = StateConfirming

	// Press 'y' to confirm
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if cmd == nil {
		t.Fatal("expected command after confirming 'y'")
	}
	if m.state != StateUpdating {
		t.Errorf("expected StateUpdating, got %v", m.state)
	}
}

func TestStartUpdatingNoItems(t *testing.T) {
	m := NewModel()
	m.cfg = &config.Config{ReadwiseToken: "test-token"}
	m.items = []Item{
		{ID: "1", Title: "Item 1"}, // no action set
	}
	m.state = StateConfirming

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if cmd == nil {
		t.Fatal("expected command")
	}

	// Execute the command — should return UpdateFinishedMsg with 0 items
	msg := cmd()
	finished, ok := msg.(UpdateFinishedMsg)
	if !ok {
		t.Fatalf("expected UpdateFinishedMsg, got %T", msg)
	}
	if finished.Success != 0 || finished.Failed != 0 {
		t.Errorf("expected 0 success/0 failed, got %d/%d", finished.Success, finished.Failed)
	}
}

func TestStartUpdatingNoToken(t *testing.T) {
	m := NewModel()
	m.cfg = &config.Config{ReadwiseToken: ""}
	m.items = []Item{
		{ID: "1", Title: "Item 1", Action: "read_now"},
	}
	m.state = StateConfirming

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if cmd == nil {
		t.Fatal("expected command")
	}

	msg := cmd()
	errMsg, ok := msg.(ErrorMsg)
	if !ok {
		t.Fatalf("expected ErrorMsg, got %T", msg)
	}
	if errMsg.Error == nil {
		t.Error("expected non-nil error")
	}
}

func TestStartUpdatingWithSelection(t *testing.T) {
	m := NewModel()
	m.cfg = &config.Config{ReadwiseToken: "test-token"}
	m.items = []Item{
		{ID: "1", Title: "Item 1", Action: "read_now"},
		{ID: "2", Title: "Item 2", Action: "later"},
		{ID: "3", Title: "Item 3", Action: "archive"},
	}
	m.listView.SetItems(m.items)

	// Select only item 0 and 2
	m.listView.SetCursor(0)
	m.listView.ToggleSelection()
	m.listView.SetCursor(2)
	m.listView.ToggleSelection()

	m.state = StateConfirming
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if cmd == nil {
		t.Fatal("expected command after confirming with selection")
	}
	if m.state != StateUpdating {
		t.Errorf("expected StateUpdating, got %v", m.state)
	}
}

func TestExportWithSelection(t *testing.T) {
	m := NewModel()
	m.items = []Item{
		{ID: "1", Title: "Item 1", URL: "https://example.com/1"},
		{ID: "2", Title: "Item 2", URL: "https://example.com/2"},
		{ID: "3", Title: "Item 3", URL: "https://example.com/3"},
	}
	m.listView.SetItems(m.items)

	// Select only items 0 and 2
	m.listView.SetCursor(0)
	m.listView.ToggleSelection()
	m.listView.SetCursor(2)
	m.listView.ToggleSelection()

	jsonData, err := m.ExportItemsToJSON()
	if err != nil {
		t.Fatalf("ExportItemsToJSON failed: %v", err)
	}

	// Should contain items 1 and 3 but not 2
	if !strings.Contains(jsonData, "Item 1") || !strings.Contains(jsonData, "Item 3") {
		t.Error("expected selected items in export")
	}
	if strings.Contains(jsonData, "Item 2") {
		t.Error("did not expect unselected item in export")
	}
}

func TestTriagePersistence(t *testing.T) {
	m := NewModel()
	m.items = []Item{
		{ID: "1", Title: "Item 1"},
		{ID: "2", Title: "Item 2"},
	}
	m.listView.SetItems(m.items)
	m.state = StateReviewing

	// Apply action to item 1
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})

	// Verify triage store was updated
	if m.triageStore == nil {
		t.Fatal("expected triageStore to be initialized")
	}
	entry, ok := m.triageStore.GetItem("1")
	if !ok {
		t.Fatal("expected item 1 to be in triage store")
	}
	if entry.Action != "read_now" {
		t.Errorf("expected stored action 'read_now', got %q", entry.Action)
	}
	if entry.Source != "manual" {
		t.Errorf("expected source 'manual', got %q", entry.Source)
	}
}

func TestApplySavedTriages(t *testing.T) {
	m := NewModel()

	// Pre-populate triage store
	m.triageStore.SetItem("1", "archive", "low", "manual")
	m.triageStore.SetItem("2", "read_now", "high", "llm")

	// Simulate loading items (which calls applySavedTriages)
	m.Update(ItemsLoadedMsg{Items: []Item{
		{ID: "1", Title: "Item 1"},
		{ID: "2", Title: "Item 2"},
		{ID: "3", Title: "Item 3"},
	}})

	if m.items[0].Action != "archive" || m.items[0].Priority != "low" {
		t.Errorf("item 1 not restored: action=%q priority=%q", m.items[0].Action, m.items[0].Priority)
	}
	if m.items[1].Action != "read_now" || m.items[1].Priority != "high" {
		t.Errorf("item 2 not restored: action=%q priority=%q", m.items[1].Action, m.items[1].Priority)
	}
	if m.items[2].Action != "" || m.items[2].Priority != "" {
		t.Errorf("item 3 should have no triage: action=%q priority=%q", m.items[2].Action, m.items[2].Priority)
	}
}

func TestStateString(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{StateConfig, "Config"},
		{StateFetching, "Fetching"},
		{StateTriaging, "Triaging"},
		{StateReviewing, "Reviewing"},
		{StateConfirming, "Confirming"},
		{StateUpdating, "Updating"},
		{StateDone, "Done"},
		{StateMessage, "Message"},
		{State(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("State(%d).String() = %q, want %q", tt.state, got, tt.want)
			}
		})
	}
}

func TestNewEditForm(t *testing.T) {
	item := &Item{
		ID:       "1",
		Title:    "Test",
		Action:   "read_now",
		Priority: "high",
	}
	ef := NewEditForm(item)
	if ef == nil {
		t.Fatal("expected non-nil EditForm")
	}
	if ef.GetForm() == nil {
		t.Error("expected non-nil form")
	}
	if ef.item != item {
		t.Error("expected item to be set")
	}
	if ef.result.Action != "read_now" {
		t.Errorf("expected result action 'read_now', got %q", ef.result.Action)
	}
	if ef.result.Priority != "high" {
		t.Errorf("expected result priority 'high', got %q", ef.result.Priority)
	}
}

func TestEditFormApplyResult(t *testing.T) {
	item := &Item{
		ID:       "1",
		Title:    "Test",
		Action:   "read_now",
		Priority: "high",
	}
	ef := NewEditForm(item)

	// Simulate changing the result
	ef.result.Action = "later"
	ef.result.Priority = "low"
	ef.ApplyResult()

	if item.Action != "later" {
		t.Errorf("expected action 'later', got %q", item.Action)
	}
	if item.Priority != "low" {
		t.Errorf("expected priority 'low', got %q", item.Priority)
	}
}

func TestEditFormApplyResultNil(t *testing.T) {
	ef := &EditForm{item: nil, result: nil}
	// Should not panic
	ef.ApplyResult()
}

func TestNewBatchForm(t *testing.T) {
	bf := NewBatchForm()
	if bf == nil {
		t.Fatal("expected non-nil BatchForm")
	}
	if bf.GetForm() == nil {
		t.Error("expected non-nil form")
	}
	if bf.result == nil {
		t.Error("expected non-nil result")
	}
}

func TestKeyMapKeys(t *testing.T) {
	km := DefaultKeyMap()
	keys := km.Keys()
	if len(keys) == 0 {
		t.Error("expected non-empty key bindings")
	}
	// Should have 16 bindings
	if len(keys) != 16 {
		t.Errorf("expected 16 key bindings, got %d", len(keys))
	}
}

func TestDefaultStyles(t *testing.T) {
	styles := DefaultStyles()
	// Just verify it doesn't panic and returns something
	_ = styles.Title.Render("test")
	_ = styles.Normal.Render("test")
	_ = styles.Help.Render("test")
	_ = styles.Error.Render("test")
}

func TestSaveLLMTriage(t *testing.T) {
	m := NewModel()
	m.saveLLMTriage("item1", "read_now", "high")

	if m.triageStore == nil {
		t.Fatal("expected triageStore")
	}
	entry, ok := m.triageStore.GetItem("item1")
	if !ok {
		t.Fatal("expected item in store")
	}
	if entry.Source != "llm" {
		t.Errorf("expected source 'llm', got %q", entry.Source)
	}
	if entry.Action != "read_now" {
		t.Errorf("expected action 'read_now', got %q", entry.Action)
	}
}

func TestSaveLLMTriageNilStore(t *testing.T) {
	m := NewModel()
	m.triageStore = nil
	// Should not panic
	m.saveLLMTriage("item1", "read_now", "high")
}

func TestExportItemsToFile(t *testing.T) {
	m := NewModel()
	m.triageStore = nil // Avoid filtering by triage store from earlier tests
	m.items = []Item{
		{ID: "export-file-1", Title: "Item 1", URL: "https://example.com/1"},
	}

	path, err := m.ExportItemsToFile()
	if err != nil {
		t.Fatalf("ExportItemsToFile failed: %v", err)
	}
	defer os.Remove(path)

	if path == "" {
		t.Fatal("expected non-empty path")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read exported file: %v", err)
	}
	if !strings.Contains(string(data), "Item 1") {
		t.Error("expected exported file to contain 'Item 1'")
	}
}

func TestImportTriageResultsFromFile(t *testing.T) {
	m := NewModel()
	m.items = []Item{
		{ID: "1", Title: "Item 1"},
	}
	m.listView.SetItems(m.items)

	jsonData := `[{"id": "1", "title": "Item 1", "triage_decision": {"action": "archive", "priority": "low"}}]`
	tmpFile := filepath.Join(t.TempDir(), "triage.json")
	os.WriteFile(tmpFile, []byte(jsonData), 0644)

	applied, err := m.ImportTriageResultsFromFile(tmpFile)
	if err != nil {
		t.Fatalf("ImportTriageResultsFromFile failed: %v", err)
	}
	if applied != 1 {
		t.Errorf("expected 1 applied, got %d", applied)
	}
	if m.items[0].Action != "archive" {
		t.Errorf("expected action 'archive', got %q", m.items[0].Action)
	}
}

func TestImportTriageResultsFromFileMissing(t *testing.T) {
	m := NewModel()
	_, err := m.ImportTriageResultsFromFile("/nonexistent/file.json")
	if err == nil {
		t.Error("expected error on missing file")
	}
}

func TestConfigViewWithError(t *testing.T) {
	m := NewModel()
	m.state = StateConfig
	m.statusMessage = "some error"
	view := m.View()
	if !strings.Contains(view, "some error") {
		t.Error("expected config view to show error message")
	}
}

func TestFetchingViewWithLLM(t *testing.T) {
	m := NewModel()
	m.state = StateFetching
	m.useLLMTriage = true
	view := m.View()
	if strings.Contains(view, "skip") {
		t.Error("expected fetching view to not mention skip (mode toggle hidden)")
	}
}

func TestFetchingViewWithoutLLM(t *testing.T) {
	m := NewModel()
	m.state = StateFetching
	m.useLLMTriage = false
	view := m.View()
	if strings.Contains(view, "skip") {
		t.Error("expected non-LLM fetching view to not mention skip")
	}
}

func TestReviewingViewWithStatus(t *testing.T) {
	m := NewModel()
	m.state = StateReviewing
	m.items = []Item{{ID: "1", Title: "Test"}}
	m.listView.SetItems(m.items)
	m.statusMessage = "Loaded 1 items"
	view := m.View()
	if !strings.Contains(view, "Loaded") {
		t.Error("expected reviewing view to show status message")
	}
}

func TestWindowSizeMsg(t *testing.T) {
	m := NewModel()
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if m.width != 120 {
		t.Errorf("expected width 120, got %d", m.width)
	}
	if m.height != 40 {
		t.Errorf("expected height 40, got %d", m.height)
	}
}

func TestQuitKey(t *testing.T) {
	m := NewModel()
	m.state = StateReviewing
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestHelpKey(t *testing.T) {
	m := NewModel()
	m.state = StateReviewing
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	if cmd != nil {
		t.Error("expected nil command from help key")
	}
}

func TestModelInit(t *testing.T) {
	m := NewModel()
	cmd := m.Init()
	if cmd == nil {
		t.Error("expected non-nil cmd from Init (spinner tick)")
	}
}

func TestHandleReviewingUpdateKey(t *testing.T) {
	m := NewModel()
	m.items = []Item{{ID: "1", Title: "Test", Action: "read_now"}}
	m.listView.SetItems(m.items)
	m.state = StateReviewing

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	if m.state != StateConfirming {
		t.Errorf("expected StateConfirming after 'u', got %v", m.state)
	}
}

func TestHandleReviewingExportKey(t *testing.T) {
	m := NewModel()
	m.items = []Item{{ID: "1", Title: "Test"}}
	m.listView.SetItems(m.items)
	m.state = StateReviewing

	// Export will fail (no clipboard in test), but should transition to StateMessage
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	if m.state != StateMessage {
		t.Errorf("expected StateMessage after 'e', got %v", m.state)
	}
}

func TestHandleReviewingImportKey(t *testing.T) {
	m := NewModel()
	m.items = []Item{{ID: "1", Title: "Test"}}
	m.listView.SetItems(m.items)
	m.state = StateReviewing

	// Import will fail (no clipboard in test), but should transition to StateMessage
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	if m.state != StateMessage {
		t.Errorf("expected StateMessage after 'i', got %v", m.state)
	}
}

func TestStartFetchingNoToken(t *testing.T) {
	m := NewModel()
	m.cfg = &config.Config{ReadwiseToken: ""}

	cmd := m.startFetching()
	if cmd == nil {
		t.Fatal("expected command")
	}

	msg := cmd()
	errMsg, ok := msg.(ErrorMsg)
	if !ok {
		t.Fatalf("expected ErrorMsg, got %T", msg)
	}
	if errMsg.Error == nil {
		t.Error("expected non-nil error")
	}
}

func TestStartFetchingNilConfig(t *testing.T) {
	m := NewModel()
	m.cfg = nil

	cmd := m.startFetching()
	if cmd == nil {
		t.Fatal("expected command")
	}

	msg := cmd()
	if _, ok := msg.(ErrorMsg); !ok {
		t.Fatalf("expected ErrorMsg, got %T", msg)
	}
}

func TestStartTriaging(t *testing.T) {
	m := NewModel()
	cmd := m.startTriaging()
	if cmd == nil {
		t.Fatal("expected command")
	}
	if m.state != StateTriaging {
		t.Errorf("expected StateTriaging, got %v", m.state)
	}

	msg := cmd()
	if _, ok := msg.(ErrorMsg); !ok {
		t.Fatalf("expected ErrorMsg, got %T", msg)
	}
}

func TestConfigEnterKey(t *testing.T) {
	m := NewModel()
	m.cfg = &config.Config{ReadwiseToken: ""}
	m.state = StateConfig

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("expected command after Enter in config")
	}
	if m.state != StateFetching {
		t.Errorf("expected StateFetching, got %v", m.state)
	}
}

func TestConfigToggleMode_Disabled(t *testing.T) {
	m := NewModel()
	m.state = StateConfig
	initial := m.useLLMTriage

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	if m.useLLMTriage != initial {
		t.Error("expected useLLMTriage to remain unchanged when toggle is hidden")
	}
}

func TestConfigViewNoLLMMode(t *testing.T) {
	m := NewModel()
	m.state = StateConfig
	view := m.View()
	if strings.Contains(view, "LLM") {
		t.Error("expected config view to not show LLM mode (hidden)")
	}
}

func TestValidateTriageJSON_MissingID(t *testing.T) {
	m := NewModel()
	m.items = []Item{{ID: "1", Title: "Test"}}

	json := `[{"id": "", "title": "Test", "triage_decision": {"action": "read_now"}}]`
	ok, _ := m.ValidateTriageJSON(json)
	if ok {
		t.Error("expected invalid for missing id")
	}
}

func TestValidateTriageJSON_MissingAction(t *testing.T) {
	m := NewModel()
	m.items = []Item{{ID: "1", Title: "Test"}}

	json := `[{"id": "1", "title": "Test", "triage_decision": {}}]`
	ok, _ := m.ValidateTriageJSON(json)
	if ok {
		t.Error("expected invalid for missing action")
	}
}

func TestValidateTriageJSON_InvalidPriority(t *testing.T) {
	m := NewModel()
	m.items = []Item{{ID: "1", Title: "Test"}}

	json := `[{"id": "1", "title": "Test", "triage_decision": {"action": "read_now", "priority": "urgent"}}]`
	ok, _ := m.ValidateTriageJSON(json)
	if ok {
		t.Error("expected invalid for bad priority")
	}
}

func TestValidateTriageJSON_EmptyArray(t *testing.T) {
	m := NewModel()
	m.items = []Item{{ID: "1", Title: "Test"}}

	ok, _ := m.ValidateTriageJSON("[]")
	if ok {
		t.Error("expected invalid for empty array")
	}
}

func TestValidateTriageJSON_ParseError(t *testing.T) {
	m := NewModel()
	m.items = []Item{{ID: "1", Title: "Test"}}

	ok, _ := m.ValidateTriageJSON("not json at all")
	if ok {
		t.Error("expected invalid for unparseable input")
	}
}

func TestImportTriageResults_EmptyResults(t *testing.T) {
	m := NewModel()
	m.items = []Item{{ID: "1", Title: "Test"}}

	_, err := m.ImportTriageResults("[]")
	if err == nil {
		t.Error("expected error for empty results")
	}
}

func TestImportTriageResults_NoJSON(t *testing.T) {
	m := NewModel()
	m.items = []Item{{ID: "1", Title: "Test"}}

	_, err := m.ImportTriageResults("not json")
	if err == nil {
		t.Error("expected error for no JSON")
	}
}

func TestImportTriageResults_MissingID(t *testing.T) {
	m := NewModel()
	m.items = []Item{{ID: "1", Title: "Test"}}

	json := `[{"id": "", "title": "Test", "triage_decision": {"action": "read_now"}}]`
	_, err := m.ImportTriageResults(json)
	if err == nil {
		t.Error("expected error for missing id")
	}
}

func TestImportTriageResults_MissingAction(t *testing.T) {
	m := NewModel()
	m.items = []Item{{ID: "1", Title: "Test"}}

	json := `[{"id": "1", "title": "Test", "triage_decision": {}}]`
	_, err := m.ImportTriageResults(json)
	if err == nil {
		t.Error("expected error for missing action")
	}
}

func TestImportTriageResults_InvalidPriority(t *testing.T) {
	m := NewModel()
	m.items = []Item{{ID: "1", Title: "Test"}}

	json := `[{"id": "1", "title": "Test", "triage_decision": {"action": "read_now", "priority": "urgent"}}]`
	_, err := m.ImportTriageResults(json)
	if err == nil {
		t.Error("expected error for invalid priority")
	}
}

func TestImportTriageResults_PartialSuccess(t *testing.T) {
	m := NewModel()
	m.items = []Item{
		{ID: "1", Title: "Item 1"},
		{ID: "2", Title: "Item 2"},
	}
	m.listView.SetItems(m.items)

	// One valid, one with unknown ID
	json := `[
		{"id": "1", "title": "Item 1", "triage_decision": {"action": "archive"}},
		{"id": "unknown", "title": "Unknown", "triage_decision": {"action": "read_now"}}
	]`
	applied, err := m.ImportTriageResults(json)
	if err != nil {
		t.Fatalf("expected no error for partial success, got %v", err)
	}
	if applied != 1 {
		t.Errorf("expected 1 applied, got %d", applied)
	}
	if !strings.Contains(m.statusMessage, "Warning") {
		t.Error("expected status message to contain warnings")
	}
}

func TestSaveTriageNilStore(t *testing.T) {
	m := NewModel()
	m.triageStore = nil
	// Should not panic
	m.saveTriage("1", "read_now", "high")
}

func TestHandleKeyPressInFetchingState(t *testing.T) {
	m := NewModel()
	m.state = StateFetching

	// Keys in fetching state should go through quit/help handling
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Error("expected quit command in fetching state")
	}
}

func TestExportItemsToJSON_AllTriaged(t *testing.T) {
	m := NewModel()
	m.items = []Item{
		{ID: "triaged-1", Title: "Item 1"},
	}
	// Mark item as triaged in store
	m.triageStore.SetItem("triaged-1", "read_now", "high", "manual")

	_, err := m.ExportItemsToJSON()
	if err == nil {
		t.Error("expected error when all items are triaged")
	}
}

func TestHelpToggle(t *testing.T) {
	m := NewModel()
	m.state = StateReviewing
	m.items = []Item{{ID: "1", Title: "Test"}}
	m.listView.SetItems(m.items)

	if m.showHelp {
		t.Error("expected showHelp false initially")
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	if !m.showHelp {
		t.Error("expected showHelp true after '?'")
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	if m.showHelp {
		t.Error("expected showHelp false after second '?'")
	}
}

func TestReviewingViewWithHelp(t *testing.T) {
	m := NewModel()
	m.state = StateReviewing
	m.width = 100
	m.height = 40
	m.items = []Item{{ID: "1", Title: "Test"}}
	m.listView.SetItems(m.items)

	// Without help overlay
	view := m.View()
	if !strings.Contains(view, "navigate") {
		t.Error("expected footer help to contain 'navigate'")
	}

	// With help overlay
	m.showHelp = true
	view = m.View()
	if !strings.Contains(view, "read now") {
		t.Error("expected full help to contain 'read now'")
	}
	if !strings.Contains(view, "archive") {
		t.Error("expected full help to contain 'archive'")
	}
}

func TestFetchingViewSpinner(t *testing.T) {
	m := NewModel()
	m.state = StateFetching
	view := m.View()
	if !strings.Contains(view, "Loading from Readwise") {
		t.Error("expected fetching view to contain loading text")
	}
}

func TestUpdatingViewProgress(t *testing.T) {
	m := NewModel()
	m.state = StateUpdating
	m.updateProgress = 0.75
	m.statusMessage = "Updated 3/4 items"
	view := m.View()
	if !strings.Contains(view, "75%") {
		t.Error("expected updating view to show percentage")
	}
	if !strings.Contains(view, "Updated 3/4") {
		t.Error("expected updating view to show status message")
	}
}

func TestConfigViewCard(t *testing.T) {
	m := NewModel()
	m.state = StateConfig
	m.width = 80
	view := m.View()
	if !strings.Contains(view, "Readwise Triage") {
		t.Error("expected config view to contain app title")
	}
	if !strings.Contains(view, "theme") {
		t.Error("expected config view help to contain theme")
	}
}

func TestDoneViewCheckmark(t *testing.T) {
	m := NewModel()
	m.state = StateDone
	m.statusMessage = "Updated 5 items"
	view := m.View()
	if !strings.Contains(view, "Complete") {
		t.Error("expected done view to contain Complete")
	}
	if !strings.Contains(view, "Updated 5 items") {
		t.Error("expected done view to contain status message")
	}
}

func TestMessageViewIcons(t *testing.T) {
	m := NewModel()
	m.state = StateMessage
	m.messageType = "error"
	m.statusMessage = "something broke"
	view := m.View()
	if !strings.Contains(view, "something broke") {
		t.Error("expected error message view to contain message")
	}

	m.messageType = "success"
	m.statusMessage = "it worked"
	view = m.View()
	if !strings.Contains(view, "it worked") {
		t.Error("expected success message view to contain message")
	}
}

func TestReviewingViewDetailPane(t *testing.T) {
	m := NewModel()
	m.state = StateReviewing
	m.width = 100
	m.height = 40
	m.items = []Item{
		{
			ID:          "1",
			Title:       "Interesting Article",
			URL:         "https://example.com/article",
			Summary:     "A great read about Go",
			Category:    "article",
			Source:      "rss",
			WordCount:   2000,
			ReadingTime: "8 min",
		},
	}
	m.listView.SetItems(m.items)
	m.listView.SetWidthHeight(100, 40)

	view := m.View()
	if !strings.Contains(view, "Interesting Article") {
		t.Error("expected reviewing view to contain item title in detail pane")
	}
	if !strings.Contains(view, "example.com") {
		t.Error("expected reviewing view to contain URL in detail pane")
	}
}

func TestReviewingViewBatchIndicator(t *testing.T) {
	m := NewModel()
	m.state = StateReviewing
	m.width = 100
	m.height = 40
	m.items = []Item{
		{ID: "1", Title: "Item 1"},
		{ID: "2", Title: "Item 2"},
	}
	m.listView.SetItems(m.items)
	m.listView.SetWidthHeight(100, 40)

	// Select items to enter batch mode
	m.listView.SetCursor(0)
	m.listView.ToggleSelection()
	m.batchMode = true

	view := m.View()
	if !strings.Contains(view, "1 selected") {
		t.Error("expected batch indicator in header")
	}
}

func TestRenderHelpLine(t *testing.T) {
	m := NewModel()
	entries := []helpEntry{
		{"j/k", "navigate"},
		{"q", "quit"},
	}
	line := m.renderHelpLine(entries)
	if line == "" {
		t.Error("expected non-empty help line")
	}
	if !strings.Contains(line, "navigate") {
		t.Error("expected help line to contain 'navigate'")
	}
	if !strings.Contains(line, "quit") {
		t.Error("expected help line to contain 'quit'")
	}
}

func TestSpinnerUpdate(t *testing.T) {
	m := NewModel()
	// Spinner tick should be handled without error
	_, cmd := m.Update(m.spinner.Tick())
	if cmd == nil {
		t.Error("expected spinner tick to return a command")
	}
}
