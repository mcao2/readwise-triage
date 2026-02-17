package triage

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// ParseTriageResponse extracts JSON array from LLM response and parses it
func ParseTriageResponse(content string) ([]Result, error) {
	// Try to find JSON array in the response
	jsonStr := extractJSON(content)
	if jsonStr == "" {
		preview := content
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		return nil, fmt.Errorf("no JSON array found in response: %s", preview)
	}

	var results []Result
	jsonStr = fixTrailingCommas(jsonStr)
	if err := json.Unmarshal([]byte(jsonStr), &results); err != nil {
		return nil, fmt.Errorf("failed to parse triage results: %w", err)
	}

	// Validate required fields
	for i, result := range results {
		if result.ID == "" {
			return nil, fmt.Errorf("result %d: missing id", i)
		}
		if result.Title == "" {
			return nil, fmt.Errorf("result %d: missing title", i)
		}
		if result.TriageDecision.Action == "" {
			return nil, fmt.Errorf("result %d: missing triage_decision.action", i)
		}
	}

	return results, nil
}

// extractJSON finds the first valid JSON array in the content
func extractJSON(content string) string {
	// Look for JSON array between triple backticks
	codeBlockRegex := regexp.MustCompile("```(?:json)?\\s*([\\s\\S]*?)```")
	matches := codeBlockRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		trimmed := strings.TrimSpace(matches[1])
		if IsJSONArray(trimmed) {
			return trimmed
		}
	}

	// Try each '[' position until we find one that yields valid JSON
	searchFrom := 0
	for searchFrom < len(content) {
		startIdx := strings.Index(content[searchFrom:], "[")
		if startIdx == -1 {
			return ""
		}
		startIdx += searchFrom

		// Find matching closing bracket
		depth := 0
		endIdx := -1
		for i := startIdx; i < len(content); i++ {
			switch content[i] {
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

		candidate := strings.TrimSpace(content[startIdx : endIdx+1])
		// Quick validation: try to unmarshal as JSON array (fix trailing commas first)
		fixed := fixTrailingCommas(candidate)
		var arr []json.RawMessage
		if json.Unmarshal([]byte(fixed), &arr) == nil && len(arr) > 0 {
			return candidate
		}

		// Not valid JSON, try next '[' after this position
		searchFrom = startIdx + 1
	}

	return ""
}

// fixTrailingCommas removes trailing commas before } or ] that LLMs sometimes produce.
func fixTrailingCommas(s string) string {
	re := regexp.MustCompile(`,\s*([}\]])`)
	return re.ReplaceAllString(s, "$1")
}

// IsJSONArray checks if the string starts with [ and ends with ]
func IsJSONArray(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]")
}

// ParseSummary extracts summary sections from the LLM response
func ParseSummary(content string) Summary {
	var summary Summary

	summary.TodayTop3 = extractSection(content, "Today's Top 3")
	summary.QuickWins = extractSection(content, "Quick Wins")
	summary.BatchDelete = extractSection(content, "Batch Delete")

	return summary
}

// extractSection extracts content between a header and the next header or end
func extractSection(content, sectionName string) []string {
	// Find the section header
	headerPattern := "**" + sectionName + "**"
	startIdx := strings.Index(content, headerPattern)
	if startIdx == -1 {
		// Try lowercase
		headerPattern = "**" + strings.ToLower(sectionName) + "**"
		startIdx = strings.Index(strings.ToLower(content), headerPattern)
	}
	if startIdx == -1 {
		return nil
	}

	// Move past the header
	startIdx += len(headerPattern)

	// Find the next section header or end of content
	sectionEnd := len(content)
	otherSections := []string{"**Today's Top 3**", "**Quick Wins**", "**Batch Delete**"}
	for _, other := range otherSections {
		if idx := strings.Index(content[startIdx:], other); idx != -1 {
			if startIdx+idx < sectionEnd {
				sectionEnd = startIdx + idx
			}
		}
	}

	sectionContent := content[startIdx:sectionEnd]
	return extractListItems(sectionContent)
}

// extractListItems extracts items from a markdown list
func extractListItems(content string) []string {
	var items []string
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Match list items: - item, * item, 1. item
		if matched, _ := regexp.MatchString(`^[\*\-\d]\.?\s+`, line); matched {
			item := regexp.MustCompile(`^[\*\-\d]\.?\s+`).ReplaceAllString(line, "")
			if item != "" {
				items = append(items, item)
			}
		}
	}
	return items
}

// Summary contains the additional summary sections from the LLM response
type Summary struct {
	TodayTop3   []string
	QuickWins   []string
	BatchDelete []string
}
