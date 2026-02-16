package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mcao2/readwise-triage/internal/triage"
	"gopkg.in/yaml.v3"
)

func TestTriageStore(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("READWISE_TRIAGE_CONFIG", filepath.Join(tmpDir, "config.yaml"))

	store, err := LoadTriageStore()
	if err != nil {
		t.Fatalf("LoadTriageStore failed: %v", err)
	}
	defer store.Close()

	store.SetItem("item1", "read_now", "high", "manual", nil, nil)
	store.SetItem("item2", "later", "medium", "llm", nil, nil)

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

	// Reopen and verify persistence
	store.Close()
	store2, err := LoadTriageStore()
	if err != nil {
		t.Fatalf("LoadTriageStore after close failed: %v", err)
	}
	defer store2.Close()

	if !store2.HasTriaged("item1") {
		t.Error("expected item1 to be triaged after reopen")
	}

	entry2, _ := store2.GetItem("item2")
	if entry2.Action != "later" {
		t.Errorf("expected action later, got %s", entry2.Action)
	}
}

func TestTriageStoreEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("READWISE_TRIAGE_CONFIG", filepath.Join(tmpDir, "config.yaml"))

	store, err := LoadTriageStore()
	if err != nil {
		t.Fatalf("LoadTriageStore failed: %v", err)
	}
	defer store.Close()

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

func TestTriageStoreWithReport(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("READWISE_TRIAGE_CONFIG", filepath.Join(tmpDir, "config.yaml"))

	store, err := LoadTriageStore()
	if err != nil {
		t.Fatalf("LoadTriageStore failed: %v", err)
	}
	defer store.Close()

	report := &triage.Result{
		ID:    "item1",
		Title: "Test Article",
		TriageDecision: triage.TriageDecision{
			Action:   "read_now",
			Priority: "high",
			Reason:   "Very relevant to current work",
		},
		ContentAnalysis: triage.ContentAnalysis{
			Type:      "tutorial",
			KeyTopics: []string{"golang", "sqlite"},
		},
		MetadataEnhancement: triage.MetadataEnhancement{
			SuggestedTags: []string{"golang", "database"},
		},
	}

	store.SetItem("item1", "read_now", "high", "llm", []string{"golang", "database"}, report)

	entry, ok := store.GetItem("item1")
	if !ok {
		t.Fatal("expected to get item1")
	}
	if entry.Report == nil {
		t.Fatal("expected non-nil report")
	}
	if entry.Report.TriageDecision.Reason != "Very relevant to current work" {
		t.Errorf("expected reason preserved, got %q", entry.Report.TriageDecision.Reason)
	}
	if len(entry.Report.ContentAnalysis.KeyTopics) != 2 {
		t.Errorf("expected 2 key topics, got %d", len(entry.Report.ContentAnalysis.KeyTopics))
	}
	if len(entry.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(entry.Tags))
	}
}

func TestTriageStoreUpsert(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("READWISE_TRIAGE_CONFIG", filepath.Join(tmpDir, "config.yaml"))

	store, err := LoadTriageStore()
	if err != nil {
		t.Fatalf("LoadTriageStore failed: %v", err)
	}
	defer store.Close()

	store.SetItem("item1", "later", "low", "manual", nil, nil)
	store.SetItem("item1", "read_now", "high", "llm", []string{"updated"}, nil)

	entry, ok := store.GetItem("item1")
	if !ok {
		t.Fatal("expected to get item1")
	}
	if entry.Action != "read_now" {
		t.Errorf("expected action read_now after upsert, got %s", entry.Action)
	}
	if entry.Priority != "high" {
		t.Errorf("expected priority high after upsert, got %s", entry.Priority)
	}
	if entry.Source != "llm" {
		t.Errorf("expected source llm after upsert, got %s", entry.Source)
	}
}

func TestTriageStoreSaveNoop(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("READWISE_TRIAGE_CONFIG", filepath.Join(tmpDir, "config.yaml"))

	store, err := LoadTriageStore()
	if err != nil {
		t.Fatalf("LoadTriageStore failed: %v", err)
	}
	defer store.Close()

	if err := store.Save(); err != nil {
		t.Errorf("Save() should be a no-op, got error: %v", err)
	}
}

func TestTriageStoreMigrateFromJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "readwise-triage")
	os.MkdirAll(configDir, 0755)
	t.Setenv("READWISE_TRIAGE_CONFIG", filepath.Join(configDir, "config.yaml"))

	// Write a legacy JSON store
	legacy := map[string]interface{}{
		"version":    "1.0",
		"updated_at": "2025-01-01T00:00:00Z",
		"items": map[string]interface{}{
			"doc1": map[string]interface{}{
				"action":     "read_now",
				"priority":   "high",
				"tags":       []string{"golang"},
				"triaged_at": "2025-01-01T00:00:00Z",
				"source":     "llm",
			},
			"doc2": map[string]interface{}{
				"action":     "archive",
				"priority":   "low",
				"triaged_at": "2025-01-01T00:00:00Z",
				"source":     "manual",
			},
		},
	}
	data, _ := json.MarshalIndent(legacy, "", "  ")
	jsonPath := filepath.Join(configDir, "triage_store.json")
	os.WriteFile(jsonPath, data, 0600)

	// Open store — should auto-migrate
	store, err := LoadTriageStore()
	if err != nil {
		t.Fatalf("LoadTriageStore failed: %v", err)
	}
	defer store.Close()

	// Verify migrated entries
	entry1, ok := store.GetItem("doc1")
	if !ok {
		t.Fatal("expected doc1 to be migrated")
	}
	if entry1.Action != "read_now" {
		t.Errorf("expected action read_now, got %s", entry1.Action)
	}
	if entry1.Priority != "high" {
		t.Errorf("expected priority high, got %s", entry1.Priority)
	}
	if len(entry1.Tags) != 1 || entry1.Tags[0] != "golang" {
		t.Errorf("expected tags [golang], got %v", entry1.Tags)
	}

	entry2, ok := store.GetItem("doc2")
	if !ok {
		t.Fatal("expected doc2 to be migrated")
	}
	if entry2.Action != "archive" {
		t.Errorf("expected action archive, got %s", entry2.Action)
	}

	// Verify JSON file was renamed to .bak
	if _, err := os.Stat(jsonPath); !os.IsNotExist(err) {
		t.Error("expected triage_store.json to be renamed")
	}
	bakPath := jsonPath + ".bak"
	if _, err := os.Stat(bakPath); os.IsNotExist(err) {
		t.Error("expected triage_store.json.bak to exist")
	}
}

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfgData := Config{
		ReadwiseToken:    "test-token",
		PerplexityAPIKey: "pplx-key",
		InboxDaysAgo:     14,
		Theme:            "dracula",
		UseLLMTriage:     true,
	}
	data, _ := yaml.Marshal(cfgData)
	os.WriteFile(configPath, data, 0600)

	t.Setenv("READWISE_TRIAGE_CONFIG", configPath)
	t.Setenv("READWISE_TOKEN", "")
	t.Setenv("PERPLEXITY_API_KEY", "")
	t.Setenv("INBOX_DAYS_AGO", "")
	t.Setenv("DEFAULT_DAYS_AGO", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.ReadwiseToken != "test-token" {
		t.Errorf("expected token 'test-token', got %q", cfg.ReadwiseToken)
	}
	if cfg.PerplexityAPIKey != "pplx-key" {
		t.Errorf("expected api key 'pplx-key', got %q", cfg.PerplexityAPIKey)
	}
	if cfg.InboxDaysAgo != 14 {
		t.Errorf("expected days 14, got %d", cfg.InboxDaysAgo)
	}
	if cfg.Theme != "dracula" {
		t.Errorf("expected theme 'dracula', got %q", cfg.Theme)
	}
}

func TestLoadConfigEnvOverride(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfgData := Config{
		ReadwiseToken: "file-token",
		InboxDaysAgo:  7,
	}
	data, _ := yaml.Marshal(cfgData)
	os.WriteFile(configPath, data, 0600)

	t.Setenv("READWISE_TRIAGE_CONFIG", configPath)
	t.Setenv("READWISE_TOKEN", "env-token")
	t.Setenv("PERPLEXITY_API_KEY", "env-pplx")
	t.Setenv("INBOX_DAYS_AGO", "30")
	t.Setenv("DEFAULT_DAYS_AGO", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.ReadwiseToken != "env-token" {
		t.Errorf("expected env token override, got %q", cfg.ReadwiseToken)
	}
	if cfg.PerplexityAPIKey != "env-pplx" {
		t.Errorf("expected env api key override, got %q", cfg.PerplexityAPIKey)
	}
	if cfg.InboxDaysAgo != 30 {
		t.Errorf("expected env days override 30, got %d", cfg.InboxDaysAgo)
	}
}

func TestLoadConfigNoFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("READWISE_TRIAGE_CONFIG", filepath.Join(tmpDir, "nonexistent.yaml"))
	t.Setenv("READWISE_TOKEN", "")
	t.Setenv("PERPLEXITY_API_KEY", "")
	t.Setenv("INBOX_DAYS_AGO", "")
	t.Setenv("DEFAULT_DAYS_AGO", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load should not fail on missing file: %v", err)
	}
	if cfg.InboxDaysAgo != 7 {
		t.Errorf("expected default days 7, got %d", cfg.InboxDaysAgo)
	}
}

func TestLoadConfigInvalidDaysAgo(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("READWISE_TRIAGE_CONFIG", filepath.Join(tmpDir, "nonexistent.yaml"))
	t.Setenv("READWISE_TOKEN", "")
	t.Setenv("PERPLEXITY_API_KEY", "")
	t.Setenv("INBOX_DAYS_AGO", "")
	t.Setenv("DEFAULT_DAYS_AGO", "not-a-number")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	// Should keep default since env value is invalid
	if cfg.InboxDaysAgo != 7 {
		t.Errorf("expected default days 7 on invalid env, got %d", cfg.InboxDaysAgo)
	}
}

func TestConfigSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	t.Setenv("READWISE_TRIAGE_CONFIG", configPath)

	cfg := &Config{
		InboxDaysAgo: 14,
		Theme:        "nord",
		UseLLMTriage: true,
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Reload and verify
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read saved config: %v", err)
	}

	var loaded Config
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to parse saved config: %v", err)
	}

	if loaded.InboxDaysAgo != 14 {
		t.Errorf("expected days 14, got %d", loaded.InboxDaysAgo)
	}
	if loaded.Theme != "nord" {
		t.Errorf("expected theme 'nord', got %q", loaded.Theme)
	}
	if !loaded.UseLLMTriage {
		t.Error("expected UseLLMTriage true")
	}
}

func TestConfigSavePreservesTokens(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	t.Setenv("READWISE_TRIAGE_CONFIG", configPath)

	// Write initial config with token
	initial := Config{
		ReadwiseToken: "my-secret-token",
		InboxDaysAgo:  7,
		Theme:         "default",
	}
	data, _ := yaml.Marshal(initial)
	os.WriteFile(configPath, data, 0600)

	// Save with different theme (should preserve token)
	cfg := &Config{
		InboxDaysAgo: 14,
		Theme:        "catppuccin",
		UseLLMTriage: false,
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Reload and check token is preserved
	savedData, _ := os.ReadFile(configPath)
	var loaded Config
	yaml.Unmarshal(savedData, &loaded)

	if loaded.ReadwiseToken != "my-secret-token" {
		t.Errorf("expected token preserved, got %q", loaded.ReadwiseToken)
	}
	if loaded.Theme != "catppuccin" {
		t.Errorf("expected theme 'catppuccin', got %q", loaded.Theme)
	}
}

func TestSaveExampleConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	t.Setenv("READWISE_TRIAGE_CONFIG", configPath)

	if err := SaveExampleConfig(); err != nil {
		t.Fatalf("SaveExampleConfig failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("expected config file to be created")
	}

	// Call again — should not overwrite
	if err := SaveExampleConfig(); err != nil {
		t.Fatalf("SaveExampleConfig second call failed: %v", err)
	}
}

func TestGetConfigPath(t *testing.T) {
	// With env var
	t.Setenv("READWISE_TRIAGE_CONFIG", "/custom/path/config.yaml")
	if got := getConfigPath(); got != "/custom/path/config.yaml" {
		t.Errorf("expected custom path, got %q", got)
	}

	// Without env var — should fall back to home dir
	t.Setenv("READWISE_TRIAGE_CONFIG", "")
	path := getConfigPath()
	if path == "" {
		t.Error("expected non-empty default config path")
	}
}

func TestGetConfigDir(t *testing.T) {
	t.Setenv("READWISE_TRIAGE_CONFIG", "/some/dir/config.yaml")
	dir, err := GetConfigDir()
	if err != nil {
		t.Fatalf("GetConfigDir failed: %v", err)
	}
	if dir != "/some/dir" {
		t.Errorf("expected '/some/dir', got %q", dir)
	}
}

func TestEnsureConfigDir(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "subdir", "config.yaml")
	t.Setenv("READWISE_TRIAGE_CONFIG", configPath)

	dir, err := EnsureConfigDir()
	if err != nil {
		t.Fatalf("EnsureConfigDir failed: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("config dir does not exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	os.WriteFile(configPath, []byte("readwise_token:\n\t- broken\n  mixed: indent"), 0600)

	t.Setenv("READWISE_TRIAGE_CONFIG", configPath)
	t.Setenv("READWISE_TOKEN", "")
	t.Setenv("PERPLEXITY_API_KEY", "")
	t.Setenv("INBOX_DAYS_AGO", "")
	t.Setenv("DEFAULT_DAYS_AGO", "")

	_, err := Load()
	if err == nil {
		t.Error("expected error on invalid YAML")
	}
}
