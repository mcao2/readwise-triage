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
		{
			name:    "multiple code blocks returns last",
			content: "Some text\n```json\n[{\"id\": \"1\"}]\n```\nMore text\n```json\n[{\"id\": \"2\"}]\n```",
			want:    `[{"id": "2"}]`,
			wantErr: false,
		},
		{
			name:    "LLM response with nested arrays in objects",
			content: "Here's the triage results:\n```json\n[\n  {\n    \"id\": \"123\",\n    \"title\": \"Test\",\n    \"triage_decision\": {\"action\": \"read_now\"},\n    \"content_analysis\": {\"key_topics\": [\"topic1\"]}\n  }\n]\n```\n\n**Today's Top 3**: item1, item2",
			want:    "[\n  {\n    \"id\": \"123\",\n    \"title\": \"Test\",\n    \"triage_decision\": {\"action\": \"read_now\"},\n    \"content_analysis\": {\"key_topics\": [\"topic1\"]}\n  }\n]",
			wantErr: false,
		},
		{
			name:    "no code blocks plain array at end",
			content: "Some explanation\n[{\"id\": \"1\"}]",
			want:    `[{"id": "1"}]`,
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

	exportData, err := m.ExportItemsToJSON()
	if err != nil {
		t.Fatalf("ExportItemsToJSON() unexpected error: %v", err)
	}

	if len(exportData) == 0 {
		t.Fatal("exported data is empty")
	}

	jsonData := extractJSONArray(exportData)
	if jsonData == "" {
		t.Fatalf("failed to extract JSON from exported data. Export starts with: %.200s", exportData)
	}

	var items []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &items); err != nil {
		t.Fatalf("Failed to parse exported JSON: %v. JSON was: %.200s", err, jsonData)
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

func TestImportTriageResults_WithDelete(t *testing.T) {
	m := &Model{
		items: []Item{
			{ID: "1", Title: "Item 1"},
		},
	}
	m.listView = NewListView(80, 20)
	m.listView.SetItems(m.items)

	jsonData := `[
		{
			"id": "1",
			"title": "Item 1",
			"triage_decision": {
				"action": "delete",
				"priority": "low"
			}
		}
	]`

	applied, err := m.ImportTriageResults(jsonData)
	if err != nil {
		t.Fatalf("ImportTriageResults failed: %v", err)
	}
	if applied != 1 {
		t.Errorf("expected 1 item applied, got %d", applied)
	}

	if m.items[0].Action != "delete" {
		t.Errorf("expected action 'delete', got %s", m.items[0].Action)
	}
}
