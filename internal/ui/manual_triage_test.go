package ui

import (
	"encoding/json"
	"testing"
)

func TestExtractJSONArray(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
		wantErr bool
	}{
		{
			name:    "plain json array",
			content: `[{"id": "1"}, {"id": "2"}]`,
			want:    `[{"id": "1"}, {"id": "2"}]`,
			wantErr: false,
		},
		{
			name:    "json in code block with json language",
			content: "```json\n[{\"id\": \"1\"}]\n```",
			want:    `[{"id": "1"}]`,
			wantErr: false,
		},
		{
			name:    "json in plain code block",
			content: "```\n[{\"id\": \"1\"}]\n```",
			want:    `[{"id": "1"}]`,
			wantErr: false,
		},
		{
			name:    "json in markdown with text before and after",
			content: "Here is the JSON:\n```json\n[{\"id\": \"1\", \"title\": \"Test\"}]\n```\nEnd of message",
			want:    `[{"id": "1", "title": "Test"}]`,
			wantErr: false,
		},
		{
			name:    "no json",
			content: "just text",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty content",
			content: "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "nested objects",
			content: `[{"id": "1", "triage_decision": {"action": "read_now", "priority": "high"}}]`,
			want:    `[{"id": "1", "triage_decision": {"action": "read_now", "priority": "high"}}]`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSONArray(tt.content)
			if (got == "") != tt.wantErr {
				if tt.wantErr {
					t.Errorf("extractJSONArray() expected error, got %q", got)
				} else {
					t.Errorf("extractJSONArray() expected %q, got %q", tt.want, got)
				}
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("extractJSONArray() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExportItemsToJSON(t *testing.T) {
	m := &Model{
		items: []Item{
			{
				ID:          "123",
				Title:       "Test Article",
				URL:         "https://example.com",
				Summary:     "A test summary",
				Category:    "article",
				Source:      "web",
				WordCount:   500,
				ReadingTime: "5 min",
			},
			{
				ID:          "456",
				Title:       "Another Article",
				URL:         "https://example2.com",
				Summary:     "Another summary",
				Category:    "article",
				Source:      "web",
				WordCount:   300,
				ReadingTime: "3 min",
			},
		},
	}

	jsonData, err := m.ExportItemsToJSON()
	if err != nil {
		t.Fatalf("ExportItemsToJSON() unexpected error: %v", err)
	}

	var items []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &items); err != nil {
		t.Fatalf("Failed to parse exported JSON: %v", err)
	}

	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}

	if items[0]["id"] != "123" {
		t.Errorf("expected first item id to be '123', got %v", items[0]["id"])
	}

	if items[0]["title"] != "Test Article" {
		t.Errorf("expected first item title to be 'Test Article', got %v", items[0]["title"])
	}

	if items[0]["word_count"] != float64(500) {
		t.Errorf("expected word_count to be 500, got %v", items[0]["word_count"])
	}
}

func TestValidActionsAndPriorities(t *testing.T) {
	if !validActions["read_now"] {
		t.Error("read_now should be a valid action")
	}
	if !validActions["later"] {
		t.Error("later should be a valid action")
	}
	if !validActions["archive"] {
		t.Error("archive should be a valid action")
	}
	if !validActions["delete"] {
		t.Error("delete should be a valid action")
	}
	if validActions["invalid"] {
		t.Error("invalid should not be a valid action")
	}

	if !validPriorities["high"] {
		t.Error("high should be a valid priority")
	}
	if !validPriorities["medium"] {
		t.Error("medium should be a valid priority")
	}
	if !validPriorities["low"] {
		t.Error("low should be a valid priority")
	}
	if validPriorities["invalid"] {
		t.Error("invalid should not be a valid priority")
	}
}
