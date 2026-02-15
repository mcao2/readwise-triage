package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestTriageStore(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("READWISE_TRIAGE_CONFIG", filepath.Join(tmpDir, "test_triage.json"))

	store, err := LoadTriageStore()
	if err != nil {
		t.Fatalf("LoadTriageStore failed: %v", err)
	}

	if store.Version != "1.0" {
		t.Errorf("expected version 1.0, got %s", store.Version)
	}

	store.SetItem("item1", "read_now", "high", "manual", nil)
	store.SetItem("item2", "later", "medium", "llm", nil)

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

	t.Setenv("READWISE_TRIAGE_CONFIG", filepath.Join(tmpDir, "empty_test.json"))

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

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfgData := Config{
		ReadwiseToken:    "test-token",
		PerplexityAPIKey: "pplx-key",
		DefaultDaysAgo:   14,
		Theme:            "dracula",
		UseLLMTriage:     true,
	}
	data, _ := yaml.Marshal(cfgData)
	os.WriteFile(configPath, data, 0600)

	t.Setenv("READWISE_TRIAGE_CONFIG", configPath)
	t.Setenv("READWISE_TOKEN", "")
	t.Setenv("PERPLEXITY_API_KEY", "")
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
	if cfg.DefaultDaysAgo != 14 {
		t.Errorf("expected days 14, got %d", cfg.DefaultDaysAgo)
	}
	if cfg.Theme != "dracula" {
		t.Errorf("expected theme 'dracula', got %q", cfg.Theme)
	}
}

func TestLoadConfigEnvOverride(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfgData := Config{
		ReadwiseToken:  "file-token",
		DefaultDaysAgo: 7,
	}
	data, _ := yaml.Marshal(cfgData)
	os.WriteFile(configPath, data, 0600)

	t.Setenv("READWISE_TRIAGE_CONFIG", configPath)
	t.Setenv("READWISE_TOKEN", "env-token")
	t.Setenv("PERPLEXITY_API_KEY", "env-pplx")
	t.Setenv("DEFAULT_DAYS_AGO", "30")

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
	if cfg.DefaultDaysAgo != 30 {
		t.Errorf("expected env days override 30, got %d", cfg.DefaultDaysAgo)
	}
}

func TestLoadConfigNoFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("READWISE_TRIAGE_CONFIG", filepath.Join(tmpDir, "nonexistent.yaml"))
	t.Setenv("READWISE_TOKEN", "")
	t.Setenv("PERPLEXITY_API_KEY", "")
	t.Setenv("DEFAULT_DAYS_AGO", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load should not fail on missing file: %v", err)
	}
	if cfg.DefaultDaysAgo != 7 {
		t.Errorf("expected default days 7, got %d", cfg.DefaultDaysAgo)
	}
}

func TestLoadConfigInvalidDaysAgo(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("READWISE_TRIAGE_CONFIG", filepath.Join(tmpDir, "nonexistent.yaml"))
	t.Setenv("READWISE_TOKEN", "")
	t.Setenv("PERPLEXITY_API_KEY", "")
	t.Setenv("DEFAULT_DAYS_AGO", "not-a-number")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	// Should keep default since env value is invalid
	if cfg.DefaultDaysAgo != 7 {
		t.Errorf("expected default days 7 on invalid env, got %d", cfg.DefaultDaysAgo)
	}
}

func TestConfigSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	t.Setenv("READWISE_TRIAGE_CONFIG", configPath)

	cfg := &Config{
		DefaultDaysAgo: 14,
		Theme:          "nord",
		UseLLMTriage:   true,
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

	if loaded.DefaultDaysAgo != 14 {
		t.Errorf("expected days 14, got %d", loaded.DefaultDaysAgo)
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
		ReadwiseToken:  "my-secret-token",
		DefaultDaysAgo: 7,
		Theme:          "default",
	}
	data, _ := yaml.Marshal(initial)
	os.WriteFile(configPath, data, 0600)

	// Save with different theme (should preserve token)
	cfg := &Config{
		DefaultDaysAgo: 14,
		Theme:          "catppuccin",
		UseLLMTriage:   false,
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

func TestTriageStoreSetItemNilMap(t *testing.T) {
	store := &TriageStore{}
	// Items is nil — SetItem should initialize it
	store.SetItem("1", "read_now", "high", "manual", nil)
	if store.Items == nil {
		t.Fatal("expected Items to be initialized")
	}
	entry, ok := store.GetItem("1")
	if !ok {
		t.Fatal("expected item to exist")
	}
	if entry.Action != "read_now" {
		t.Errorf("expected action 'read_now', got %q", entry.Action)
	}
}

func TestTriageStoreGetItemNilMap(t *testing.T) {
	store := &TriageStore{}
	_, ok := store.GetItem("anything")
	if ok {
		t.Error("expected false from nil map")
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	os.WriteFile(configPath, []byte("readwise_token:\n\t- broken\n  mixed: indent"), 0600)

	t.Setenv("READWISE_TRIAGE_CONFIG", configPath)
	t.Setenv("READWISE_TOKEN", "")
	t.Setenv("PERPLEXITY_API_KEY", "")
	t.Setenv("DEFAULT_DAYS_AGO", "")

	_, err := Load()
	if err == nil {
		t.Error("expected error on invalid YAML")
	}
}

func TestLoadTriageStoreCorruptedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "readwise-triage")
	os.MkdirAll(configDir, 0755)
	storePath := filepath.Join(configDir, "triage_store.json")
	os.WriteFile(storePath, []byte("{not valid json}"), 0600)

	t.Setenv("READWISE_TRIAGE_CONFIG", filepath.Join(configDir, "config.yaml"))

	_, err := LoadTriageStore()
	if err == nil {
		t.Error("expected error on corrupted JSON")
	}
}

func TestTriageStoreSaveUpdatesTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("READWISE_TRIAGE_CONFIG", filepath.Join(tmpDir, "config.yaml"))

	store := &TriageStore{
		Version: "1.0",
		Items:   make(map[string]TriageEntry),
	}
	store.SetItem("1", "read_now", "high", "manual", nil)

	if err := store.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if store.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set after save")
	}
}

func TestLoadTriageStoreNilItems(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "readwise-triage")
	os.MkdirAll(configDir, 0755)
	storePath := filepath.Join(configDir, "triage_store.json")
	// Valid JSON but with null items
	os.WriteFile(storePath, []byte(`{"version":"1.0","items":null}`), 0600)

	t.Setenv("READWISE_TRIAGE_CONFIG", filepath.Join(configDir, "config.yaml"))

	store, err := LoadTriageStore()
	if err != nil {
		t.Fatalf("LoadTriageStore failed: %v", err)
	}
	if store.Items == nil {
		t.Error("expected Items to be initialized even when null in JSON")
	}
}
