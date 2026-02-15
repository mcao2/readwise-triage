package config

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/mcao2/readwise-triage/internal/triage"
)

// TriageEntry represents a single triaged item.
type TriageEntry struct {
	Action    string
	Priority  string
	Tags      []string
	Source    string
	TriagedAt string
	Report   *triage.Result // full LLM report, nil for manual entries
}

// TriageStore persists triage decisions in a SQLite database.
type TriageStore struct {
	db *sql.DB
}

func getTriageDBPath() string {
	configDir, err := EnsureConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(configDir, "triage.db")
}

// LoadTriageStore opens (or creates) the SQLite-backed triage store.
// If a legacy triage_store.json exists, its entries are migrated automatically.
func LoadTriageStore() (*TriageStore, error) {
	dbPath := getTriageDBPath()
	if dbPath == "" {
		return nil, fmt.Errorf("cannot determine triage store path")
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open triage db: %w", err)
	}

	// Enable WAL mode for better concurrent read performance.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	createSQL := `CREATE TABLE IF NOT EXISTS triage_entries (
		id         TEXT PRIMARY KEY,
		action     TEXT NOT NULL,
		priority   TEXT NOT NULL DEFAULT '',
		tags       TEXT,
		source     TEXT NOT NULL,
		triaged_at TEXT NOT NULL,
		report     TEXT
	)`
	if _, err := db.Exec(createSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("create table: %w", err)
	}

	store := &TriageStore{db: db}

	// Auto-migrate from legacy JSON file.
	if err := store.migrateFromJSON(); err != nil {
		// Migration failure is non-fatal — log-style: we just skip.
		_ = err
	}

	return store, nil
}

// Close closes the underlying database connection.
func (s *TriageStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// SetItem upserts a triage entry. report may be nil for manual entries.
func (s *TriageStore) SetItem(id, action, priority, source string, tags []string, report *triage.Result) {
	var tagsJSON *string
	if len(tags) > 0 {
		b, _ := json.Marshal(tags)
		str := string(b)
		tagsJSON = &str
	}

	var reportJSON *string
	if report != nil {
		b, _ := json.Marshal(report)
		str := string(b)
		reportJSON = &str
	}

	now := time.Now().Format(time.RFC3339)

	_, _ = s.db.Exec(`INSERT INTO triage_entries (id, action, priority, tags, source, triaged_at, report)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			action=excluded.action,
			priority=excluded.priority,
			tags=excluded.tags,
			source=excluded.source,
			triaged_at=excluded.triaged_at,
			report=excluded.report`,
		id, action, priority, tagsJSON, source, now, reportJSON)
}

// GetItem retrieves a triage entry by document ID.
func (s *TriageStore) GetItem(id string) (TriageEntry, bool) {
	row := s.db.QueryRow(
		`SELECT action, priority, tags, source, triaged_at, report FROM triage_entries WHERE id = ?`, id)

	var entry TriageEntry
	var tagsJSON, reportJSON sql.NullString

	if err := row.Scan(&entry.Action, &entry.Priority, &tagsJSON, &entry.Source, &entry.TriagedAt, &reportJSON); err != nil {
		return TriageEntry{}, false
	}

	if tagsJSON.Valid {
		_ = json.Unmarshal([]byte(tagsJSON.String), &entry.Tags)
	}
	if reportJSON.Valid {
		var r triage.Result
		if json.Unmarshal([]byte(reportJSON.String), &r) == nil {
			entry.Report = &r
		}
	}

	return entry, true
}

// HasTriaged returns true if the given document ID has been triaged.
func (s *TriageStore) HasTriaged(id string) bool {
	var exists int
	err := s.db.QueryRow(`SELECT 1 FROM triage_entries WHERE id = ?`, id).Scan(&exists)
	return err == nil
}

// GetUntriagedIDs returns the subset of allIDs that have not been triaged.
func (s *TriageStore) GetUntriagedIDs(allIDs []string) []string {
	var result []string
	for _, id := range allIDs {
		if !s.HasTriaged(id) {
			result = append(result, id)
		}
	}
	return result
}

// Save is a no-op retained for caller compatibility. Writes are immediate.
func (s *TriageStore) Save() error {
	return nil
}

// legacyTriageStore mirrors the old JSON file structure for migration.
type legacyTriageStore struct {
	Version   string                  `json:"version"`
	UpdatedAt time.Time               `json:"updated_at"`
	Items     map[string]legacyEntry  `json:"items"`
}

type legacyEntry struct {
	Action    string   `json:"action"`
	Priority  string   `json:"priority"`
	Tags      []string `json:"tags,omitempty"`
	TriagedAt string   `json:"triaged_at"`
	Source    string   `json:"source"`
}

func (s *TriageStore) migrateFromJSON() error {
	configDir, err := EnsureConfigDir()
	if err != nil {
		return err
	}
	jsonPath := filepath.Join(configDir, "triage_store.json")

	data, err := os.ReadFile(jsonPath)
	if os.IsNotExist(err) {
		return nil // nothing to migrate
	}
	if err != nil {
		return fmt.Errorf("read legacy store: %w", err)
	}

	var legacy legacyTriageStore
	if err := json.Unmarshal(data, &legacy); err != nil {
		return fmt.Errorf("parse legacy store: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin migration tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO triage_entries (id, action, priority, tags, source, triaged_at, report)
		VALUES (?, ?, ?, ?, ?, ?, NULL)`)
	if err != nil {
		return fmt.Errorf("prepare migration stmt: %w", err)
	}
	defer stmt.Close()

	for id, entry := range legacy.Items {
		var tagsJSON *string
		if len(entry.Tags) > 0 {
			b, _ := json.Marshal(entry.Tags)
			str := string(b)
			tagsJSON = &str
		}
		source := entry.Source
		if source == "" {
			source = "manual"
		}
		triagedAt := entry.TriagedAt
		if triagedAt == "" {
			triagedAt = time.Now().Format(time.RFC3339)
		}
		if _, err := stmt.Exec(id, entry.Action, entry.Priority, tagsJSON, source, triagedAt); err != nil {
			return fmt.Errorf("migrate entry %s: %w", id, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration: %w", err)
	}

	// Rename the old file so we don't re-migrate.
	backupPath := jsonPath + ".bak"
	_ = os.Rename(jsonPath, backupPath)

	return nil
}
