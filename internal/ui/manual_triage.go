package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/mcao2/readwise-triage/internal/triage"
)

// Valid actions for triage decisions
var validActions = map[string]bool{
	"read_now": true,
	"later":    true,
	"archive":  true,
	"delete":   true,
}

// Valid priorities for triage decisions
var validPriorities = map[string]bool{
	"high":   true,
	"medium": true,
	"low":    true,
}

// ExportItemsToJSON exports the current items with triage prompt for manual LLM triage
func (m *Model) ExportItemsToJSON() (string, error) {
	type exportItem struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		URL         string `json:"url"`
		Summary     string `json:"summary"`
		Category    string `json:"category"`
		Source      string `json:"source"`
		WordCount   int    `json:"word_count"`
		ReadingTime string `json:"reading_time"`
	}

	items := make([]exportItem, len(m.items))
	for i, item := range m.items {
		items[i] = exportItem{
			ID:          item.ID,
			Title:       item.Title,
			URL:         item.URL,
			Summary:     item.Summary,
			Category:    item.Category,
			Source:      item.Source,
			WordCount:   item.WordCount,
			ReadingTime: item.ReadingTime,
		}
	}

	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal items: %w", err)
	}

	promptPart := triage.PromptTemplate
	idx := strings.LastIndex(promptPart, "**待处理的 inbox 条目：**")
	if idx == -1 {
		return string(data), nil
	}

	output := promptPart[:idx+len("**待处理的 inbox 条目：**\n\n")]
	output += "```json\n"
	output += string(data)
	output += "\n```"

	return output, nil
}

// ExportItemsToClipboard exports items to clipboard
func (m *Model) ExportItemsToClipboard() error {
	jsonData, err := m.ExportItemsToJSON()
	if err != nil {
		return err
	}

	if err := clipboard.WriteAll(jsonData); err != nil {
		return fmt.Errorf("failed to copy to clipboard: %w", err)
	}

	return nil
}

// ExportItemsToFile exports items to a temp file and returns the path
func (m *Model) ExportItemsToFile() (string, error) {
	jsonData, err := m.ExportItemsToJSON()
	if err != nil {
		return "", err
	}

	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "readwise-export.json")

	counter := 1
	for {
		if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
			break
		}
		tmpFile = filepath.Join(tmpDir, fmt.Sprintf("readwise-export-%d.json", counter))
		counter++
	}

	if err := os.WriteFile(tmpFile, []byte(jsonData), 0644); err != nil {
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	return tmpFile, nil
}

// ImportTriageResultsFromClipboard reads from clipboard and imports triage results
func (m *Model) ImportTriageResultsFromClipboard() (int, error) {
	data, err := clipboard.ReadAll()
	if err != nil {
		return 0, fmt.Errorf("failed to read clipboard: %w", err)
	}

	if strings.TrimSpace(data) == "" {
		return 0, fmt.Errorf("clipboard is empty")
	}

	return m.ImportTriageResults(data)
}

// ImportTriageResultsFromFile reads from a file and imports triage results
func (m *Model) ImportTriageResultsFromFile(filePath string) (int, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read file: %w", err)
	}

	return m.ImportTriageResults(string(data))
}

// ImportTriageResults parses and validates triage results JSON and applies to items
func (m *Model) ImportTriageResults(jsonData string) (int, error) {
	// Extract JSON array from the content (handle markdown code blocks)
	jsonStr := extractJSONArray(jsonData)
	if jsonStr == "" {
		return 0, fmt.Errorf("no valid JSON array found in input")
	}

	var results []triage.Result
	if err := json.Unmarshal([]byte(jsonStr), &results); err != nil {
		return 0, fmt.Errorf("failed to parse JSON: %w", err)
	}

	if len(results) == 0 {
		return 0, fmt.Errorf("empty results array")
	}

	// Validate and apply results
	applied := 0
	errors := []string{}

	// Create a map for quick lookup
	itemMap := make(map[string]*Item)
	for i := range m.items {
		itemMap[m.items[i].ID] = &m.items[i]
	}

	for i, result := range results {
		// Validate required fields
		if result.ID == "" {
			errors = append(errors, fmt.Sprintf("result %d: missing id", i))
			continue
		}

		if result.Title == "" {
			errors = append(errors, fmt.Sprintf("result %d: missing title", i))
			continue
		}

		// Validate triage decision
		if result.TriageDecision.Action == "" {
			errors = append(errors, fmt.Sprintf("result %d (%s): missing triage_decision.action", i, result.Title))
			continue
		}

		if !validActions[result.TriageDecision.Action] {
			errors = append(errors, fmt.Sprintf("result %d (%s): invalid action '%s' (must be one of: read_now, later, archive, delete)", i, result.Title, result.TriageDecision.Action))
			continue
		}

		// Validate priority if provided
		if result.TriageDecision.Priority != "" && !validPriorities[result.TriageDecision.Priority] {
			errors = append(errors, fmt.Sprintf("result %d (%s): invalid priority '%s' (must be one of: high, medium, low)", i, result.Title, result.TriageDecision.Priority))
			continue
		}

		// Find and update the item
		item, ok := itemMap[result.ID]
		if !ok {
			errors = append(errors, fmt.Sprintf("result %d: id '%s' not found in items", i, result.ID))
			continue
		}

		// Apply the triage decision
		item.Action = result.TriageDecision.Action
		item.Priority = result.TriageDecision.Priority

		applied++
	}

	if applied == 0 && len(errors) > 0 {
		return 0, fmt.Errorf("validation failed:\n%s", strings.Join(errors, "\n"))
	}

	if len(errors) > 0 {
		// Partial success - return what we could apply plus warnings
		m.statusMessage = fmt.Sprintf("Applied %d/%d results. Warnings:\n%s", applied, len(results), strings.Join(errors, "\n"))
	} else {
		m.statusMessage = fmt.Sprintf("Successfully applied triage results to %d items", applied)
	}

	// Refresh the list view
	m.listView.SetItems(m.items)

	return applied, nil
}

// extractJSONArray finds and extracts JSON array from content
func extractJSONArray(content string) string {
	// Try to find JSON in code blocks first
	content = strings.TrimSpace(content)

	// Check for markdown code blocks
	if strings.Contains(content, "```") {
		// Extract content between ```json and ``` or ``` and ```
		start := strings.Index(content, "```json")
		if start == -1 {
			start = strings.Index(content, "```")
		}
		if start != -1 {
			// Skip the ``` marker
			start += 3
			// Skip language specifier if present
			for start < len(content) && (content[start] == '\n' || content[start] == ' ' || content[start] == '\r') {
				start++
			}
			if start < len(content) && (content[start] >= 'a' && content[start] <= 'z') {
				for start < len(content) && content[start] != '\n' {
					start++
				}
			}

			end := strings.Index(content[start:], "```")
			if end != -1 {
				content = content[start : start+end]
			}
		}
	}

	content = strings.TrimSpace(content)

	// Find the JSON array
	startIdx := strings.Index(content, "[")
	if startIdx == -1 {
		return ""
	}

	// Find matching closing bracket
	depth := 0
	endIdx := -1
	for i := startIdx; i < len(content); i++ {
		switch content[i] {
		case '{':
			depth++
		case '}':
			depth--
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				endIdx = i
				break
			}
		}
		if endIdx != -1 {
			break
		}
	}

	if endIdx == -1 {
		return ""
	}

	return strings.TrimSpace(content[startIdx : endIdx+1])
}

// ValidateTriageJSON validates triage results JSON without applying them
func (m *Model) ValidateTriageJSON(jsonData string) (bool, string) {
	jsonStr := extractJSONArray(jsonData)
	if jsonStr == "" {
		return false, "no valid JSON array found in input"
	}

	var results []triage.Result
	if err := json.Unmarshal([]byte(jsonStr), &results); err != nil {
		return false, fmt.Sprintf("JSON parse error: %v", err)
	}

	if len(results) == 0 {
		return false, "empty results array"
	}

	// Build a set of valid item IDs
	validIDs := make(map[string]bool)
	for _, item := range m.items {
		validIDs[item.ID] = true
	}

	// Validate each result
	errors := []string{}
	for i, result := range results {
		if result.ID == "" {
			errors = append(errors, fmt.Sprintf("result %d: missing id", i))
			continue
		}

		if !validIDs[result.ID] {
			errors = append(errors, fmt.Sprintf("result %d: unknown id '%s'", i, result.ID))
		}

		if result.TriageDecision.Action == "" {
			errors = append(errors, fmt.Sprintf("result %d (%s): missing action", i, result.Title))
		} else if !validActions[result.TriageDecision.Action] {
			errors = append(errors, fmt.Sprintf("result %d (%s): invalid action '%s'", i, result.Title, result.TriageDecision.Action))
		}

		if result.TriageDecision.Priority != "" && !validPriorities[result.TriageDecision.Priority] {
			errors = append(errors, fmt.Sprintf("result %d (%s): invalid priority '%s'", i, result.Title, result.TriageDecision.Priority))
		}
	}

	if len(errors) > 0 {
		return false, strings.Join(errors, "\n")
	}

	return true, fmt.Sprintf("valid: %d results for %d items", len(results), len(results))
}
