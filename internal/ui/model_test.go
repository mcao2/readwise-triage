package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
	if m.state != StateConfig {
		t.Errorf("expected Config state after key in Done, got %v", m.state)
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
