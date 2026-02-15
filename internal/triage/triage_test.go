package triage

import (
	"testing"
)

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

func TestParseSummary(t *testing.T) {
	content := `
**Today's Top 3**:
1. First important item
2. Second important item

**Quick Wins**:
- Quick item 1
- Quick item 2

**Batch Delete**:
- Delete item 1
`

	summary := ParseSummary(content)

	if len(summary.TodayTop3) != 2 {
		t.Errorf("expected 2 top 3 items, got %d", len(summary.TodayTop3))
	}
	if len(summary.QuickWins) != 2 {
		t.Errorf("expected 2 quick wins, got %d", len(summary.QuickWins))
	}
	if len(summary.BatchDelete) != 1 {
		t.Errorf("expected 1 batch delete, got %d", len(summary.BatchDelete))
	}
}

func TestExtractListItems(t *testing.T) {
	content := `
- Item 1
- Item 2
* Item 3
1. Item 4
2. Item 5
`

	items := extractListItems(content)

	if len(items) != 5 {
		t.Errorf("expected 5 items, got %d", len(items))
	}
	if items[0] != "Item 1" {
		t.Errorf("expected 'Item 1', got %q", items[0])
	}
}

func TestNewPerplexityClient(t *testing.T) {
	tests := []struct {
		name    string
		apiKey  string
		envKey  string
		wantErr bool
	}{
		{
			name:    "valid api key",
			apiKey:  "test-key",
			wantErr: false,
		},
		{
			name:    "empty key with env",
			apiKey:  "",
			envKey:  "env-key",
			wantErr: false,
		},
		{
			name:    "empty key no env",
			apiKey:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envKey != "" {
				t.Setenv("PERPLEXITY_API_KEY", tt.envKey)
			} else {
				t.Setenv("PERPLEXITY_API_KEY", "")
			}

			client, err := NewPerplexityClient(tt.apiKey)
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
			if client == nil {
				t.Error("expected client, got nil")
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
