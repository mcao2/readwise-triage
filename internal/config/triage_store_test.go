package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTriageStore(t *testing.T) {
	tmpDir := t.TempDir()

	os.Setenv("READWISE_TRIAGE_CONFIG", filepath.Join(tmpDir, "test_triage.json"))
	defer os.Unsetenv("READWISE_TRIAGE_CONFIG")

	store, err := LoadTriageStore()
	if err != nil {
		t.Fatalf("LoadTriageStore failed: %v", err)
	}

	if store.Version != "1.0" {
		t.Errorf("expected version 1.0, got %s", store.Version)
	}

	store.SetItem("item1", "read_now", "high", "manual")
	store.SetItem("item2", "later", "medium", "llm")

	if !store.HasTriaged("item1") {
		t.Error("expected item1 to be triaged")
	}

	if store.HasTriaged("item3") {
		t.Error("expected item3 to not be triaged")
	}

	entry, ok := store.GetItem("item1")
	if !ok {
		t.Error("expected to get item1")
	}
	if entry.Action != "read_now" {
		t.Errorf("expected action read_now, got %s", entry.Action)
	}
	if entry.Priority != "high" {
		t.Errorf("expected priority high, got %s", entry.Priority)
	}
	if entry.Source != "manual" {
		t.Errorf("expected source manual, got %s", entry.Source)
	}

	allIDs := []string{"item1", "item2", "item3", "item4"}
	untriaged := store.GetUntriagedIDs(allIDs)
	if len(untriaged) != 2 {
		t.Errorf("expected 2 untriaged items, got %d: %v", len(untriaged), untriaged)
	}

	if err := store.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	store2, err := LoadTriageStore()
	if err != nil {
		t.Fatalf("LoadTriageStore after save failed: %v", err)
	}

	if !store2.HasTriaged("item1") {
		t.Error("expected item1 to be triaged after reload")
	}

	entry2, _ := store2.GetItem("item2")
	if entry2.Action != "later" {
		t.Errorf("expected action later, got %s", entry2.Action)
	}
}

func TestTriageStoreEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	os.Setenv("READWISE_TRIAGE_CONFIG", filepath.Join(tmpDir, "empty_test.json"))
	defer os.Unsetenv("READWISE_TRIAGE_CONFIG")

	store, err := LoadTriageStore()
	if err != nil {
		t.Fatalf("LoadTriageStore failed: %v", err)
	}

	if store.HasTriaged("anything") {
		t.Error("expected empty store to have no items")
	}

	entry, ok := store.GetItem("anything")
	if ok {
		t.Errorf("expected no entry, got %+v", entry)
	}

	untriaged := store.GetUntriagedIDs([]string{"a", "b"})
	if len(untriaged) != 2 {
		t.Errorf("expected all untriaged, got %d", len(untriaged))
	}
}
