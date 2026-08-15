package triage

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// ParseTriageResponse extracts JSON from LLM response and parses it
func ParseTriageResponse(content string) ([]Result, error) {
	jsonStr := extractJSON(content)
	if jsonStr == "" {
		preview := content
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		return nil, fmt.Errorf("no JSON found in response: %s", preview)
	}

	jsonStr = fixTrailingCommas(jsonStr)

	// Try as array first
	var results []Result
	if err := json.Unmarshal([]byte(jsonStr), &results); err == nil {
		return validateResults(results)
	}

	// Try as single object (wrap in array)
	var single Result
	if err := json.Unmarshal([]byte(jsonStr), &single); err == nil {
		return validateResults([]Result{single})
	}

	// Try as object with array field (e.g., {"results": [...]})
	var wrapper map[string]json.RawMessage
	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err == nil {
		for _, key := range []string{"results", "items", "data", "triage_results"} {
			if raw, ok := wrapper[key]; ok {
				if err := json.Unmarshal(raw, &results); err == nil {
					return validateResults(results)
				}
			}
		}
	}

	return nil, fmt.Errorf("failed to parse triage results from: %s", truncate(jsonStr, 200))
}

func validateResults(results []Result) ([]Result, error) {
	if len(results) == 0 {
		return nil, fmt.Errorf("empty results array")
	}
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

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

// extractJSON finds the first valid JSON array or object in the content
func extractJSON(content string) string {
	// Look for JSON in code blocks first
	codeBlockRegex := regexp.MustCompile("```(?:json)?\\s*([\\s\\S]*?)```")
	matches := codeBlockRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		trimmed := strings.TrimSpace(matches[1])
		if strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "{") {
			return trimmed
		}
	}

	// Try to extract bare JSON (array or object)
	for _, opener := range []byte{'[', '{'} {
		if s := extractBalanced(content, opener); s != "" {
			return s
		}
	}

	return ""
}

// extractBalanced finds the first balanced JSON value starting with `opener`
func extractBalanced(content string, opener byte) string {
	searchFrom := 0
	for searchFrom < len(content) {
		startIdx := strings.IndexByte(content[searchFrom:], opener)
		if startIdx == -1 {
			return ""
		}
		startIdx += searchFrom

		closer := byte(']')
		if opener == '{' {
			closer = '}'
		}

		depth := 0
		inString := false
		escaped := false
		endIdx := -1

		for i := startIdx; i < len(content); i++ {
			c := content[i]
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' && inString {
				escaped = true
				continue
			}
			if c == '"' {
				inString = !inString
				continue
			}
			if inString {
				continue
			}
			if c == opener {
				depth++
			} else if c == closer {
				depth--
				if depth == 0 {
					endIdx = i
					break
				}
			}
		}

		if endIdx == -1 {
			return ""
		}

		candidate := strings.TrimSpace(content[startIdx : endIdx+1])
		fixed := fixTrailingCommas(candidate)
		var raw json.RawMessage
		if json.Unmarshal([]byte(fixed), &raw) == nil {
			return candidate
		}

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
