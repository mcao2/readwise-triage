package triage

import (
	"strings"
	"testing"
)

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestParseTriageResponse(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantCount int
		wantErr   bool
	}{
		{
			name: "valid json array",
			content: `[
				{
					"id": "item1",
					"title": "Test Article",
					"url": "https://example.com",
					"triage_decision": {
						"action": "read_now",
						"priority": "high",
						"reason": "Important article"
					}
				}
			]`,
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "json in code block",
			content:   "```json\n[\n  {\n    \"id\": \"item1\",\n    \"title\": \"Test\",\n    \"url\": \"https://example.com\",\n    \"triage_decision\": {\n      \"action\": \"later\",\n      \"priority\": \"medium\",\n      \"reason\": \"Good article\"\n    }\n  }\n]\n```",
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "empty content",
			content:   "",
			wantCount: 0,
			wantErr:   true,
		},
		{
			name:      "no json",
			content:   "This is just text without any JSON",
			wantCount: 0,
			wantErr:   true,
		},
		{
			name: "json with trailing commas",
			content: `[{
				"id": "item1",
				"title": "Test",
				"url": "https://example.com",
				"triage_decision": {
					"action": "archive",
					"priority": "low",
					"reason": "Not relevant",
				},
			}]`,
			wantCount: 1,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ParseTriageResponse(tt.content)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if len(results) != tt.wantCount {
				t.Errorf("expected %d results, got %d", tt.wantCount, len(results))
			}
		})
	}
}

func TestParseTriageResponseValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name: "missing id",
			content: `[{
				"title": "Test",
				"url": "https://example.com",
				"triage_decision": {"action": "read_now", "priority": "high", "reason": "test"}
			}]`,
			wantErr: "missing id",
		},
		{
			name: "missing title",
			content: `[{
				"id": "item1",
				"url": "https://example.com",
				"triage_decision": {"action": "read_now", "priority": "high", "reason": "test"}
			}]`,
			wantErr: "missing title",
		},
		{
			name: "missing action",
			content: `[{
				"id": "item1",
				"title": "Test",
				"url": "https://example.com",
				"triage_decision": {"priority": "high", "reason": "test"}
			}]`,
			wantErr: "missing triage_decision.action",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseTriageResponse(tt.content)
			if err == nil {
				t.Error("expected error, got nil")
				return
			}
			if !contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "plain json array",
			content: `[{"id": "1"}, {"id": "2"}]`,
			want:    `[{"id": "1"}, {"id": "2"}]`,
		},
		{
			name:    "json in code block",
			content: "```json\n[{\"id\": \"1\"}]\n```",
			want:    `[{"id": "1"}]`,
		},
		{
			name:    "json in plain code block",
			content: "```\n[{\"id\": \"1\"}]\n```",
			want:    `[{"id": "1"}]`,
		},
		{
			name:    "no json",
			content: "just text",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSON(tt.content)
			if got != tt.want {
				t.Errorf("extractJSON() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsJSONArray(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{"valid array", "[1, 2, 3]", true},
		{"with spaces", "  [1, 2]  ", true},
		{"empty array", "[]", true},
		{"not array", "{}", false},
		{"string", "hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsJSONArray(tt.s)
			if got != tt.want {
				t.Errorf("IsJSONArray(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestExtractJSONWithMixedContent(t *testing.T) {
	content := `Here is the analysis:

Some text before the JSON.

[{"id": "1", "title": "Test", "url": "https://example.com", "triage_decision": {"action": "read_now", "priority": "high", "reason": "test"}}]

**Today's Top 3**:
1. Item one
`
	results, err := ParseTriageResponse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestExtractJSONUnmatchedBracket(t *testing.T) {
	content := "[{\"id\": \"1\""
	result := extractJSON(content)
	if result != "" {
		t.Errorf("expected empty string for unmatched bracket, got %q", result)
	}
}
