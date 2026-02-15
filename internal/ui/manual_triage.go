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

// ExportItemsToJSON exports only untriaged items with triage prompt for manual LLM triage
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

	var items []exportItem
	selectedIndices := m.listView.GetSelected()
	useSelection := len(selectedIndices) > 0

	for i, item := range m.items {
		if useSelection {
			isSelected := false
			for _, idx := range selectedIndices {
				if idx == i {
					isSelected = true
					break
				}
			}
			if !isSelected {
				continue
			}
		} else if m.triageStore != nil && m.triageStore.HasTriaged(item.ID) {
			continue
		}

		items = append(items, exportItem{
			ID:          item.ID,
			Title:       item.Title,
			URL:         item.URL,
			Summary:     item.Summary,
			Category:    item.Category,
			Source:      item.Source,
			WordCount:   item.WordCount,
			ReadingTime: item.ReadingTime,
		})
	}

	if len(items) == 0 {
		return "", fmt.Errorf("all items have already been triaged")
	}

	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal items: %w", err)
	}

	promptPart := triage.PromptTemplate
	markers := []string{
		"**Inbox items to process:**",
		"**待处理的 inbox 条目：**",
	}
	idx := -1
	marker := ""
	for _, m := range markers {
		i := strings.LastIndex(promptPart, m)
		if i > idx {
			idx = i
			marker = m
		}
	}
	if idx == -1 {
		return string(data), nil
	}

	output := promptPart[:idx+len(marker)+2]
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

		displayTitle := result.Title
		if displayTitle == "" {
			displayTitle = result.ID
		}

		// Validate triage decision
		if result.TriageDecision.Action == "" {
			errors = append(errors, fmt.Sprintf("result %d (%s): missing triage_decision.action", i, displayTitle))
			continue
		}

		if !validActions[result.TriageDecision.Action] {
			errors = append(errors, fmt.Sprintf("result %d (%s): invalid action '%s' (must be one of: read_now, later, archive)", i, displayTitle, result.TriageDecision.Action))
			continue
		}

		// Validate priority if provided
		if result.TriageDecision.Priority != "" && !validPriorities[result.TriageDecision.Priority] {
			errors = append(errors, fmt.Sprintf("result %d (%s): invalid priority '%s' (must be one of: high, medium, low)", i, displayTitle, result.TriageDecision.Priority))
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

		// Save to triage store
		if m.triageStore != nil {
			m.triageStore.SetItem(item.ID, item.Action, item.Priority, "llm")
		}

		applied++
	}

	if applied == 0 && len(errors) > 0 {
		return 0, fmt.Errorf("validation failed:\n%s", strings.Join(errors, "\n"))
	}

	// Save triage store
	if m.triageStore != nil {
		_ = m.triageStore.Save()
	}

	if len(errors) > 0 {
		m.statusMessage = fmt.Sprintf("Applied %d/%d results. Warnings:\n%s", applied, len(results), strings.Join(errors, "\n"))
	} else {
		m.statusMessage = fmt.Sprintf("Successfully applied triage results to %d items", applied)
	}

	m.listView.SetItems(m.items)

	return applied, nil
}

func isJSONArray(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]")
}

func extractJSONArray(content string) string {
	content = strings.TrimSpace(content)

	if strings.Contains(content, "```") {
		arrays := extractJSONArraysFromCodeBlocks(content)
		if len(arrays) > 0 {
			return arrays[len(arrays)-1]
		}
	}

	arrays := extractAllJSONArrays(content)
	if len(arrays) > 0 {
		return arrays[len(arrays)-1]
	}

	return ""
}

func extractJSONArraysFromCodeBlocks(content string) []string {
	var results []string
	marker := "```"

	start := 0
	for {
		idx := strings.Index(content[start:], marker)
		if idx == -1 {
			break
		}
		idx += start
		codeStart := idx + len(marker)

		if codeStart < len(content) && content[codeStart] == 'j' {
			codeStart++
			for codeStart < len(content) && (content[codeStart] >= 'a' && content[codeStart] <= 'z') {
				codeStart++
			}
		}

		for codeStart < len(content) && (content[codeStart] == '\n' || content[codeStart] == ' ') {
			codeStart++
		}

		end := strings.Index(content[codeStart:], marker)
		if end == -1 {
			break
		}
		end += codeStart

		codeContent := strings.TrimSpace(content[codeStart:end])
		if isJSONArray(codeContent) {
			results = append(results, codeContent)
		}

		start = end + len(marker)
	}

	return results
}

func extractAllJSONArrays(content string) []string {
	var results []string

	startIdx := 0
	for {
		idx := strings.Index(content[startIdx:], "[")
		if idx == -1 {
			break
		}
		idx += startIdx

		depth := 0
		endIdx := -1
		for i := idx; i < len(content); i++ {
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

		if endIdx != -1 {
			arr := strings.TrimSpace(content[idx : endIdx+1])
			if isJSONArray(arr) {
				results = append(results, arr)
			}
			startIdx = endIdx + 1
		} else {
			break
		}
	}

	return results
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
