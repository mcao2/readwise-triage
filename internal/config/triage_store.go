package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type TriageStore struct {
	Version   string                 `json:"version"`
	UpdatedAt time.Time              `json:"updated_at"`
	Items     map[string]TriageEntry `json:"items"`
}

type TriageEntry struct {
	Action    string   `json:"action"`
	Priority  string   `json:"priority"`
	Tags      []string `json:"tags,omitempty"`
	TriagedAt string   `json:"triaged_at"`
	Source    string   `json:"source"` // "manual", "llm"
}

func GetTriageStorePath() string {
	configDir, err := EnsureConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(configDir, "triage_store.json")
}

func LoadTriageStore() (*TriageStore, error) {
	path := GetTriageStorePath()
	if path == "" {
		return &TriageStore{Items: make(map[string]TriageEntry)}, nil
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &TriageStore{Version: "1.0", Items: make(map[string]TriageEntry)}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read triage store: %w", err)
	}

	var store TriageStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("failed to parse triage store: %w", err)
	}

	if store.Items == nil {
		store.Items = make(map[string]TriageEntry)
	}

	return &store, nil
}

func (s *TriageStore) Save() error {
	path := GetTriageStorePath()
	if path == "" {
		return fmt.Errorf("cannot determine triage store path")
	}

	s.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal triage store: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

func (s *TriageStore) SetItem(id, action, priority, source string, tags []string) {
	if s.Items == nil {
		s.Items = make(map[string]TriageEntry)
	}
	s.Items[id] = TriageEntry{
		Action:    action,
		Priority:  priority,
		Tags:      tags,
		TriagedAt: time.Now().Format(time.RFC3339),
		Source:    source,
	}
}

func (s *TriageStore) GetItem(id string) (TriageEntry, bool) {
	if s.Items == nil {
		return TriageEntry{}, false
	}
	entry, ok := s.Items[id]
	return entry, ok
}

func (s *TriageStore) HasTriaged(id string) bool {
	_, ok := s.Items[id]
	return ok
}

func (s *TriageStore) GetUntriagedIDs(allIDs []string) []string {
	var result []string
	for _, id := range allIDs {
		if !s.HasTriaged(id) {
			result = append(result, id)
		}
	}
	return result
}
