package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mcao2/readwise-triage/internal/triage"
)

// TriageEntry represents a single triaged item.
type TriageEntry struct {
	Action    string         `json:"action"`
	Tags      []string       `json:"tags,omitempty"`
	Source    string         `json:"source"`
	TriagedAt string         `json:"triaged_at"`
	Report    *triage.Result `json:"report,omitempty"`
}

// TriageStore persists triage decisions in a JSON file.
type TriageStore struct {
	mu   sync.Mutex
	path string
	data map[string]TriageEntry
}

func getTriageStorePath() string {
	configDir, err := EnsureConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(configDir, "triage_store.json")
}

// LoadTriageStore opens (or creates) the JSON-backed triage store.
func LoadTriageStore() (*TriageStore, error) {
	path := getTriageStorePath()
	if path == "" {
		return nil, fmt.Errorf("cannot determine triage store path")
	}

	store := &TriageStore{
		path: path,
		data: make(map[string]TriageEntry),
	}

	if err := store.load(); err != nil {
		return nil, fmt.Errorf("load triage store: %w", err)
	}

	return store, nil
}

func (s *TriageStore) load() error {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil // fresh store
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.data)
}

func (s *TriageStore) save() error {
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

// Close is a no-op for JSON stores. Writes are immediate.
func (s *TriageStore) Close() error {
	return nil
}

// SetItem upserts a triage entry. report may be nil for manual entries.
func (s *TriageStore) SetItem(id, action, source string, tags []string, report *triage.Result) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[id] = TriageEntry{
		Action:    action,
		Tags:      tags,
		Source:    source,
		TriagedAt: time.Now().Format(time.RFC3339),
		Report:    report,
	}
	_ = s.save()
}

// GetItem retrieves a triage entry by document ID.
func (s *TriageStore) GetItem(id string) (TriageEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.data[id]
	return entry, ok
}

// HasTriaged returns true if the given document ID has been triaged.
func (s *TriageStore) HasTriaged(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.data[id]
	return ok
}

// GetUntriagedIDs returns the subset of allIDs that have not been triaged.
func (s *TriageStore) GetUntriagedIDs(allIDs []string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []string
	for _, id := range allIDs {
		if _, ok := s.data[id]; !ok {
			result = append(result, id)
		}
	}
	return result
}

// Save is a no-op retained for caller compatibility. Writes are immediate.
func (s *TriageStore) Save() error {
	return nil
}
