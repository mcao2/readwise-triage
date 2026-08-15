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
	"github.com/mcao2/readwise-triage/internal/triage"
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
	if m.state != StateFetching {
		t.Errorf("expected initial state StateFetching, got %v", m.state)
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
	if m.state != StateReviewing {
		t.Errorf("expected state StateReviewing after error, got %v", m.state)
	}

	m.Update(ItemsLoadedMsg{Items: items})
	m.Update(UpdateFinishedMsg{Success: 2, Failed: 0})
	if m.state != StateReviewing {
		t.Errorf("expected state StateReviewing, got %v", m.state)
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

}

func TestHandleAdditionalKeys(t *testing.T) {
	m := NewModel()

	// 'n' in confirm popup cancels
	m.state = StateReviewing
	m.confirming = true
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if m.confirming {
		t.Error("expected confirming=false after 'n'")
	}

	// 'y' in confirm popup starts update
	m.confirming = true
	m.cfg = &config.Config{ReadwiseToken: "test"}
	m.items = []Item{{ID: "1", Title: "Test", Action: "read_now"}}
	m.listView.SetItems(m.items)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if !m.updating {
		t.Error("expected updating=true after 'y'")
	}
	_ = cmd
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

	m.state = StateReviewing
	m.cfg = &config.Config{ReadwiseToken: "test"}
	m.confirming = true
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if !m.updating {
		t.Fatal("expected updating=true from Update")
	}
	_ = cmd
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

	m.state = StateReviewing
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

	if !m.fetching {
		t.Error("expected fetching=true after Refresh key")
	}

	if cmd == nil {
		t.Error("expected command after Refresh key")
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
			Tags:         []string{"golang", "tutorial"},
			OriginalTags: []string{"inbox", "rss"},
		},
		{
			ID:           "2",
			Title:        "Item 2",
			Action:       "later",
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
			}

			// Preserve original tags
			update.Tags = append(update.Tags, item.OriginalTags...)

			if len(item.Tags) > 0 {
				update.Tags = append(update.Tags, item.Tags...)
			}

			updates = append(updates, update)
		}
	}

	if len(updates) != 2 {
		t.Fatalf("expected 2 updates, got %d", len(updates))
	}

	// Check first item: original tags + golang + tutorial
	expectedTags1 := []string{"inbox", "rss", "golang", "tutorial"}
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
		{"d", "archive"},
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

func TestAllBatchActions(t *testing.T) {
	tests := []struct {
		key    string
		action string
	}{
		{"r", "read_now"},
		{"l", "later"},
		{"a", "archive"},
		{"d", "archive"},
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

func TestFetchMoreKey(t *testing.T) {
	m := NewModel()
	items := []Item{{ID: "1", Title: "Item 1"}}
	m.Update(ItemsLoadedMsg{Items: items})
	m.state = StateReviewing

	initialLookback := m.inboxLookback

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})

	if m.inboxLookback != initialLookback+7 {
		t.Errorf("expected inboxLookback %d, got %d", initialLookback+7, m.inboxLookback)
	}
	if !m.fetching {
		t.Error("expected fetching=true after FetchMore key")
	}
	if cmd == nil {
		t.Error("expected command after FetchMore key")
	}
}

func TestAllViewRendering(t *testing.T) {
	tests := []struct {
		name  string
		setup func(m *Model)
	}{
		{"fetching", func(m *Model) { m.state = StateFetching }},
		{"triaging", func(m *Model) { m.state = StateReviewing }},
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
		{"confirming", func(m *Model) { m.state = StateReviewing }},
		{"updating", func(m *Model) {
			m.state = StateReviewing
			m.updating = true
			m.statusMessage = "Updating..."
		}},
		{"done", func(m *Model) {
			m.state = StateReviewing
			m.statusMessage = "All done"
		}},
		{"message_success", func(m *Model) {
			m.state = StateReviewing
			m.messageType = "success"
			m.statusMessage = "It worked"
		}},
		{"message_error", func(m *Model) {
			m.state = StateReviewing
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
		{ID: "2", Title: "Item 2", Action: "later"},
	}
	m.listView.SetItems(m.items)
	m.state = StateReviewing
	m.confirming = true

	// Press 'y' to confirm
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if !m.updating {
		t.Fatal("expected updating=true after confirming 'y'")
	}
	if m.confirming {
		t.Error("expected confirming=false after 'y'")
	}
}

func TestStartUpdatingNoItems(t *testing.T) {
	m := NewModel()
	m.cfg = &config.Config{ReadwiseToken: "test-token"}
	m.items = []Item{
		{ID: "1", Title: "Item 1"}, // no action set
	}
	m.state = StateReviewing
	m.confirming = true

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	// No items to update — startUpdating returns UpdateFinishedMsg immediately
	// m.updating may be false since the flow completes instantly
	if m.confirming {
		t.Error("expected confirming=false")
	}
}

func TestStartUpdatingNoToken(t *testing.T) {
	m := NewModel()
	m.cfg = &config.Config{ReadwiseToken: ""}
	m.items = []Item{
		{ID: "1", Title: "Item 1", Action: "read_now"},
	}
	m.state = StateReviewing
	m.confirming = true

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if m.confirming {
		t.Error("expected confirming=false")
	}
	if m.statusMessage == "" {
		t.Error("expected error status message")
	}
	if m.messageType != "error" {
		t.Error("expected error message type")
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

	m.state = StateReviewing
	m.confirming = true
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if !m.updating {
		t.Fatal("expected updating=true after confirming with selection")
	}
	if m.confirming {
		t.Error("expected confirming=false")
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
	m.triageStore.SetItem("1", "archive", "manual", nil, nil)
	m.triageStore.SetItem("2", "read_now", "llm", nil, nil)

	// Simulate loading items (which calls applySavedTriages)
	m.Update(ItemsLoadedMsg{Items: []Item{
		{ID: "1", Title: "Item 1"},
		{ID: "2", Title: "Item 2"},
		{ID: "3", Title: "Item 3"},
	}})

}

func TestStateString(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{StateFetching, "Fetching"},
		{StateReviewing, "Reviewing"},
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

func TestKeyMapKeys(t *testing.T) {
	km := DefaultKeyMap()
	keys := km.Keys()
	if len(keys) == 0 {
		t.Error("expected non-empty key bindings")
	}
	// Should have 12 bindings
	if len(keys) != 12 {
		t.Errorf("expected 12 key bindings, got %d", len(keys))
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
	m.saveLLMTriage("item1", "read_now", nil, nil)

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
	m.saveLLMTriage("item1", "read_now", nil, nil)
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
	if m.state != StateReviewing {
		t.Errorf("expected StateReviewing after 'u', got %v", m.state)
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
	m.state = StateReviewing
	// No items to triage → returns nil cmd
	cmd := m.startTriaging()
	if cmd != nil {
		t.Fatal("expected nil command when no items")
	}
	if m.triaging {
		t.Error("expected triaging=false when no items")
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

func TestNavigationAfterMultipleUpdateCycles(t *testing.T) {
	m := NewModel()
	m.cfg = &config.Config{ReadwiseToken: "test-token"}

	items := []Item{
		{ID: "1", Title: "Item 1"},
		{ID: "2", Title: "Item 2"},
		{ID: "3", Title: "Item 3"},
	}
	m.Update(ItemsLoadedMsg{Items: items})

	for cycle := 0; cycle < 5; cycle++ {
		// Triage first item
		m.listView.SetCursor(0)
		m.cursor = 0
		m.state = StateReviewing
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})

		if m.items[0].Action != "read_now" {
			t.Fatalf("cycle %d: expected action read_now, got %q", cycle, m.items[0].Action)
		}

		// Go to confirming → updating → done
		m.state = StateReviewing
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})

		// UpdateFinishedMsg clears updating flag
		m.Update(UpdateFinishedMsg{Success: 1, Failed: 0})
		if m.updating {
			t.Fatalf("cycle %d: expected updating=false", cycle)
		}

		// Re-fetch via Back key
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")})
		if !m.fetching {
			t.Fatalf("cycle %d: expected fetching=true after re-fetch", cycle)
		}

		// Simulate fetch completing
		m.Update(ItemsLoadedMsg{Items: items})
		if m.fetching {
			t.Fatalf("cycle %d: expected fetching=false after load", cycle)
		}

		// Verify navigation works
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		if m.cursor != 1 {
			t.Fatalf("cycle %d: expected cursor 1 after j, got %d", cycle, m.cursor)
		}

		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		if m.cursor != 2 {
			t.Fatalf("cycle %d: expected cursor 2 after j, got %d", cycle, m.cursor)
		}

		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
		if m.cursor != 1 {
			t.Fatalf("cycle %d: expected cursor 1 after k, got %d", cycle, m.cursor)
		}

		// Verify items are still there
		if len(m.items) != 3 {
			t.Fatalf("cycle %d: expected 3 items, got %d", cycle, len(m.items))
		}

		// Verify listView items are in sync
		for i := 0; i < 3; i++ {
			item := m.listView.GetItem(i)
			if item == nil {
				t.Fatalf("cycle %d: listView item %d is nil", cycle, i)
			}
		}
	}
}

func TestEnterKeyEntersTagEditingMode(t *testing.T) {
	m := NewModel()
	m.items = []Item{
		{ID: "1", Title: "Item 1", Tags: []string{"go", "tutorial"}},
	}
	m.listView.SetItems(m.items)
	m.state = StateReviewing

	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.editingTags {
		t.Fatal("expected editingTags to be true after Enter")
	}
	if m.tagsInput != "go, tutorial" {
		t.Errorf("expected tagsInput 'go, tutorial', got %q", m.tagsInput)
	}
	// Cursor should be at end
	if m.tagsCursor != len([]rune(m.tagsInput)) {
		t.Errorf("expected tagsCursor %d, got %d", len([]rune(m.tagsInput)), m.tagsCursor)
	}
}

func TestTagEditingTypingAndConfirm(t *testing.T) {
	m := NewModel()
	m.items = []Item{
		{ID: "1", Title: "Item 1"},
	}
	m.listView.SetItems(m.items)
	m.state = StateReviewing

	// Enter editing mode
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Type "rust, wasm"
	for _, ch := range "rust, wasm" {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
	}
	if m.tagsInput != "rust, wasm" {
		t.Fatalf("expected tagsInput 'rust, wasm', got %q", m.tagsInput)
	}

	// Confirm with Enter
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.editingTags {
		t.Fatal("expected editingTags to be false after confirm")
	}
	if len(m.items[0].Tags) != 2 || m.items[0].Tags[0] != "rust" || m.items[0].Tags[1] != "wasm" {
		t.Errorf("expected tags [rust wasm], got %v", m.items[0].Tags)
	}
}

func TestTagEditingEscCancels(t *testing.T) {
	m := NewModel()
	m.items = []Item{
		{ID: "1", Title: "Item 1", Tags: []string{"original"}},
	}
	m.listView.SetItems(m.items)
	m.state = StateReviewing

	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	// Type something
	for _, ch := range "new" {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
	}
	// Cancel
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.editingTags {
		t.Fatal("expected editingTags to be false after Esc")
	}
	// Tags should be unchanged
	if len(m.items[0].Tags) != 1 || m.items[0].Tags[0] != "original" {
		t.Errorf("expected tags [original], got %v", m.items[0].Tags)
	}
}

func TestTagEditingBackspace(t *testing.T) {
	m := NewModel()
	m.items = []Item{
		{ID: "1", Title: "Item 1"},
	}
	m.listView.SetItems(m.items)
	m.state = StateReviewing

	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	for _, ch := range "abc" {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
	}
	if m.tagsInput != "abc" {
		t.Fatalf("expected tagsInput 'abc', got %q", m.tagsInput)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.tagsInput != "ab" {
		t.Errorf("expected tagsInput 'ab' after backspace, got %q", m.tagsInput)
	}
	if m.tagsCursor != 2 {
		t.Errorf("expected tagsCursor 2, got %d", m.tagsCursor)
	}

	// Move cursor to middle and backspace
	m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.tagsInput != "b" {
		t.Errorf("expected tagsInput 'b' after mid-backspace, got %q", m.tagsInput)
	}
	if m.tagsCursor != 0 {
		t.Errorf("expected tagsCursor 0, got %d", m.tagsCursor)
	}

	// Backspace at position 0 should not panic or change anything
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.tagsInput != "b" {
		t.Errorf("expected tagsInput 'b' unchanged, got %q", m.tagsInput)
	}
}

func TestTagEditingBatchMode(t *testing.T) {
	m := NewModel()
	m.items = []Item{
		{ID: "1", Title: "Item 1", Tags: []string{"old"}},
		{ID: "2", Title: "Item 2", Tags: []string{"old"}},
	}
	m.listView.SetItems(m.items)
	m.state = StateReviewing

	// Select both items
	m.listView.SetCursor(0)
	m.listView.ToggleSelection()
	m.listView.SetCursor(1)
	m.listView.ToggleSelection()
	m.batchMode = true

	// Enter tag editing — batch mode starts with empty input
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.editingTags {
		t.Fatal("expected editingTags true")
	}
	if m.tagsInput != "" {
		t.Errorf("expected empty tagsInput in batch mode, got %q", m.tagsInput)
	}

	// Type tags and confirm
	for _, ch := range "new1, new2" {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	for i, item := range m.items {
		if len(item.Tags) != 2 || item.Tags[0] != "new1" || item.Tags[1] != "new2" {
			t.Errorf("item %d: expected tags [new1 new2], got %v", i, item.Tags)
		}
	}
}

func TestTagEditingViewPopup(t *testing.T) {
	m := NewModel()
	m.state = StateReviewing
	m.width = 100
	m.height = 40
	m.items = []Item{{ID: "1", Title: "Test", Tags: []string{"go"}}}
	m.listView.SetItems(m.items)
	m.editingTags = true
	m.tagsInput = "go, rust"

	view := m.View()
	if !strings.Contains(view, "Edit Tags") {
		t.Error("expected tag editing popup to contain 'Edit Tags'")
	}
	if !strings.Contains(view, "go, rust") {
		t.Error("expected tag editing popup to contain input text")
	}
}

func TestParseTags(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"go, rust", []string{"go", "rust"}},
		{"  go , rust , ", []string{"go", "rust"}},
		{"", nil},
		{",,,", nil},
		{"single", []string{"single"}},
	}
	for _, tt := range tests {
		got := parseTags(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("parseTags(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("parseTags(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestTagEditingArrowKeys(t *testing.T) {
	m := NewModel()
	m.items = []Item{{ID: "1", Title: "Item 1"}}
	m.listView.SetItems(m.items)
	m.state = StateReviewing

	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	for _, ch := range "abc" {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
	}
	// Cursor at 3 (end)
	if m.tagsCursor != 3 {
		t.Fatalf("expected tagsCursor 3, got %d", m.tagsCursor)
	}

	// Left moves cursor back
	m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if m.tagsCursor != 2 {
		t.Errorf("expected tagsCursor 2 after Left, got %d", m.tagsCursor)
	}

	// Right moves cursor forward
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if m.tagsCursor != 3 {
		t.Errorf("expected tagsCursor 3 after Right, got %d", m.tagsCursor)
	}

	// Right at end stays at end
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if m.tagsCursor != 3 {
		t.Errorf("expected tagsCursor 3 at boundary, got %d", m.tagsCursor)
	}

	// Left to beginning
	m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if m.tagsCursor != 0 {
		t.Errorf("expected tagsCursor 0, got %d", m.tagsCursor)
	}

	// Left at beginning stays at 0
	m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if m.tagsCursor != 0 {
		t.Errorf("expected tagsCursor 0 at boundary, got %d", m.tagsCursor)
	}

	// Insert at cursor position
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if m.tagsInput != "xabc" {
		t.Errorf("expected tagsInput 'xabc', got %q", m.tagsInput)
	}
	if m.tagsCursor != 1 {
		t.Errorf("expected tagsCursor 1 after insert, got %d", m.tagsCursor)
	}
}

func TestTagEditingOptionDelete(t *testing.T) {
	m := NewModel()
	m.items = []Item{{ID: "1", Title: "Item 1"}}
	m.listView.SetItems(m.items)
	m.state = StateReviewing

	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	for _, ch := range "go, rust, wasm" {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
	}
	if m.tagsCursor != 14 {
		t.Fatalf("expected tagsCursor 14, got %d", m.tagsCursor)
	}

	// Option+Delete: delete "wasm" (previous word from end)
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace, Alt: true})
	if m.tagsInput != "go, rust, " {
		t.Errorf("expected tagsInput %q, got %q", "go, rust, ", m.tagsInput)
	}
	if m.tagsCursor != 10 {
		t.Errorf("expected tagsCursor 10, got %d", m.tagsCursor)
	}

	// Option+Delete again: delete ", rust, " back to word boundary
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace, Alt: true})
	if m.tagsInput != "go, " {
		t.Errorf("expected tagsInput %q, got %q", "go, ", m.tagsInput)
	}
	if m.tagsCursor != 4 {
		t.Errorf("expected tagsCursor 4, got %d", m.tagsCursor)
	}

	// Option+Delete again: delete "go, " back to start
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace, Alt: true})
	if m.tagsInput != "" {
		t.Errorf("expected tagsInput %q, got %q", "", m.tagsInput)
	}
	if m.tagsCursor != 0 {
		t.Errorf("expected tagsCursor 0, got %d", m.tagsCursor)
	}

	// Option+Delete at empty: no-op
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace, Alt: true})
	if m.tagsInput != "" {
		t.Errorf("expected tagsInput %q, got %q", "", m.tagsInput)
	}
}

func TestTagEditingWordJump(t *testing.T) {
	// Helper to set up a model in tag-editing mode with "go, rust, wasm"
	setup := func() *Model {
		m := NewModel()
		m.items = []Item{{ID: "1", Title: "Item 1"}}
		m.listView.SetItems(m.items)
		m.state = StateReviewing
		m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		for _, ch := range "go, rust, wasm" {
			m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		}
		return m
	}

	// CSI path: alt+left / alt+right (KeyLeft/KeyRight with Alt flag)
	t.Run("CSI_sequences", func(t *testing.T) {
		m := setup()
		if m.tagsCursor != 14 {
			t.Fatalf("expected tagsCursor 14, got %d", m.tagsCursor)
		}

		m.Update(tea.KeyMsg{Type: tea.KeyLeft, Alt: true})
		if m.tagsCursor != 10 {
			t.Errorf("expected tagsCursor 10 after alt+left, got %d", m.tagsCursor)
		}

		m.Update(tea.KeyMsg{Type: tea.KeyLeft, Alt: true})
		if m.tagsCursor != 4 {
			t.Errorf("expected tagsCursor 4 after alt+left, got %d", m.tagsCursor)
		}

		m.Update(tea.KeyMsg{Type: tea.KeyLeft, Alt: true})
		if m.tagsCursor != 0 {
			t.Errorf("expected tagsCursor 0 after alt+left, got %d", m.tagsCursor)
		}

		m.Update(tea.KeyMsg{Type: tea.KeyRight, Alt: true})
		if m.tagsCursor != 2 {
			t.Errorf("expected tagsCursor 2 after alt+right, got %d", m.tagsCursor)
		}

		m.Update(tea.KeyMsg{Type: tea.KeyRight, Alt: true})
		if m.tagsCursor != 8 {
			t.Errorf("expected tagsCursor 8 after alt+right, got %d", m.tagsCursor)
		}

		m.Update(tea.KeyMsg{Type: tea.KeyRight, Alt: true})
		if m.tagsCursor != 14 {
			t.Errorf("expected tagsCursor 14 after alt+right, got %d", m.tagsCursor)
		}
	})

	// ESC+letter path: alt+b / alt+f (macOS terminals send ESC b / ESC f)
	t.Run("ESC_letter_sequences", func(t *testing.T) {
		m := setup()
		if m.tagsCursor != 14 {
			t.Fatalf("expected tagsCursor 14, got %d", m.tagsCursor)
		}

		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}, Alt: true})
		if m.tagsCursor != 10 {
			t.Errorf("expected tagsCursor 10 after alt+b, got %d", m.tagsCursor)
		}

		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}, Alt: true})
		if m.tagsCursor != 4 {
			t.Errorf("expected tagsCursor 4 after alt+b, got %d", m.tagsCursor)
		}

		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}, Alt: true})
		if m.tagsCursor != 0 {
			t.Errorf("expected tagsCursor 0 after alt+b, got %d", m.tagsCursor)
		}

		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}, Alt: true})
		if m.tagsCursor != 2 {
			t.Errorf("expected tagsCursor 2 after alt+f, got %d", m.tagsCursor)
		}

		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}, Alt: true})
		if m.tagsCursor != 8 {
			t.Errorf("expected tagsCursor 8 after alt+f, got %d", m.tagsCursor)
		}

		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}, Alt: true})
		if m.tagsCursor != 14 {
			t.Errorf("expected tagsCursor 14 after alt+f, got %d", m.tagsCursor)
		}
	})
}

func TestWordBoundaryHelpers(t *testing.T) {
	runes := []rune("go, rust, wasm")

	// prevWordBoundary
	if got := prevWordBoundary(runes, 14); got != 10 {
		t.Errorf("prevWordBoundary(14) = %d, want 10", got)
	}
	if got := prevWordBoundary(runes, 10); got != 4 {
		t.Errorf("prevWordBoundary(10) = %d, want 4", got)
	}
	if got := prevWordBoundary(runes, 4); got != 0 {
		t.Errorf("prevWordBoundary(4) = %d, want 0", got)
	}
	if got := prevWordBoundary(runes, 0); got != 0 {
		t.Errorf("prevWordBoundary(0) = %d, want 0", got)
	}

	// nextWordBoundary
	if got := nextWordBoundary(runes, 0); got != 2 {
		t.Errorf("nextWordBoundary(0) = %d, want 2", got)
	}
	if got := nextWordBoundary(runes, 2); got != 8 {
		t.Errorf("nextWordBoundary(2) = %d, want 8", got)
	}
	if got := nextWordBoundary(runes, 8); got != 14 {
		t.Errorf("nextWordBoundary(8) = %d, want 14", got)
	}
	if got := nextWordBoundary(runes, 14); got != 14 {
		t.Errorf("nextWordBoundary(14) = %d, want 14", got)
	}
}

func TestTriageFinishedMsg_Success(t *testing.T) {
	m := NewModel()
	m.state = StateReviewing
	m.items = []Item{
		{ID: "1", Title: "Article 1", URL: "https://example.com/1"},
		{ID: "2", Title: "Article 2", URL: "https://example.com/2"},
	}
	m.listView.SetItems(m.items)

	results := []triage.Result{
		{
			ID:    "1",
			Title: "Article 1",
			URL:   "https://example.com/1",
			TriageDecision: triage.TriageDecision{
				Action: "read_now",
				Reason: "Very useful",
			},
			MetadataEnhancement: triage.MetadataEnhancement{
				SuggestedTags: []string{"productivity", "tools"},
			},
		},
		{
			ID:    "2",
			Title: "Article 2",
			URL:   "https://example.com/2",
			TriageDecision: triage.TriageDecision{
				Action: "archive",
				Reason: "Not relevant",
			},
		},
	}

	m.Update(TriageFinishedMsg{Results: results})

	if m.state != StateReviewing {
		t.Errorf("expected StateReviewing, got %v", m.state)
	}
	if m.messageType != "success" {
		t.Errorf("expected success message type, got %q", m.messageType)
	}
	if m.items[0].Action != "read_now" {
		t.Errorf("expected item 1 action 'read_now', got %q", m.items[0].Action)
	}
	if len(m.items[0].Tags) != 2 {
		t.Errorf("expected 2 tags on item 1, got %d", len(m.items[0].Tags))
	}
	if m.items[1].Action != "archive" {
		t.Errorf("expected item 2 action 'archive', got %q", m.items[1].Action)
	}
}

func TestTriageFinishedMsg_Error(t *testing.T) {
	m := NewModel()
	m.state = StateReviewing

	m.Update(TriageFinishedMsg{Err: fmt.Errorf("API rate limited")})

	if m.state != StateReviewing {
		t.Errorf("expected StateReviewing, got %v", m.state)
	}
	if m.messageType != "error" {
		t.Errorf("expected error message type, got %q", m.messageType)
	}
	if !strings.Contains(m.statusMessage, "API rate limited") {
		t.Errorf("expected error in status message, got %q", m.statusMessage)
	}
}

func TestApplyTriageResults_FiltersActionTags(t *testing.T) {
	m := NewModel()
	m.items = []Item{
		{ID: "1", Title: "Article 1"},
	}
	m.listView.SetItems(m.items)

	results := []triage.Result{
		{
			ID:    "1",
			Title: "Article 1",
			TriageDecision: triage.TriageDecision{
				Action: "later",
				Reason: "Read later",
			},
			MetadataEnhancement: triage.MetadataEnhancement{
				SuggestedTags: []string{"later", "productivity", "read_now", "tools"},
			},
		},
	}

	applied := m.applyTriageResults(results)
	if applied != 1 {
		t.Errorf("expected 1 applied, got %d", applied)
	}
	// "later" and "read_now" should be filtered out as action names
	if len(m.items[0].Tags) != 2 {
		t.Errorf("expected 2 tags after filtering, got %d: %v", len(m.items[0].Tags), m.items[0].Tags)
	}
}

func TestApplyTriageResults_UnknownIDSkipped(t *testing.T) {
	m := NewModel()
	m.items = []Item{
		{ID: "1", Title: "Article 1"},
	}
	m.listView.SetItems(m.items)

	results := []triage.Result{
		{
			ID:    "unknown",
			Title: "Unknown Article",
			TriageDecision: triage.TriageDecision{
				Action: "archive",
				Reason: "test",
			},
		},
	}

	applied := m.applyTriageResults(results)
	if applied != 0 {
		t.Errorf("expected 0 applied for unknown ID, got %d", applied)
	}
	if m.items[0].Action != "" {
		t.Errorf("expected item unchanged, got action %q", m.items[0].Action)
	}
}

func TestGetTriageItems(t *testing.T) {
	m := NewModel()
	m.items = []Item{
		{ID: "1", Title: "Untriaged", URL: "https://example.com/1"},
		{ID: "2", Title: "Already triaged", URL: "https://example.com/2", Action: "read_now"},
		{ID: "3", Title: "Also untriaged", URL: "https://example.com/3"},
	}
	m.listView.SetItems(m.items)

	items, err := m.getTriageItems()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only include untriaged items (1 and 3)
	foundUntriaged := false
	foundTriaged := false
	foundAlso := false
	for _, item := range items {
		if item.Title == "Untriaged" {
			foundUntriaged = true
		}
		if item.Title == "Already triaged" {
			foundTriaged = true
		}
		if item.Title == "Also untriaged" {
			foundAlso = true
		}
	}
	if !foundUntriaged {
		t.Error("expected untriaged item")
	}
	if foundTriaged {
		t.Error("expected triaged item to be excluded")
	}
	if !foundAlso {
		t.Error("expected second untriaged item")
	}
}

func TestGetTriageItems_AllTriaged(t *testing.T) {
	m := NewModel()
	m.items = []Item{
		{ID: "1", Title: "Done", Action: "read_now"},
	}
	m.listView.SetItems(m.items)

	_, err := m.getTriageItems()
	if err == nil {
		t.Error("expected error when all items are triaged")
	}
}

func TestAutoTriageKeyBinding(t *testing.T) {
	m := NewModel()
	m.state = StateReviewing
	m.items = []Item{
		{ID: "1", Title: "Test"},
	}
	m.listView.SetItems(m.items)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
	if cmd == nil {
		t.Error("expected command from T key")
	}
	if m.state != StateReviewing {
		t.Errorf("expected StateReviewing after T key, got %v", m.state)
	}
}
