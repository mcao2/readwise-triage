package readwise

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

// mockHTTPClient is a test double for HTTPClient
type mockHTTPClient struct {
	responses []*http.Response
	errors    []error
	callCount int
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	defer func() { m.callCount++ }()
	if m.callCount < len(m.errors) && m.errors[m.callCount] != nil {
		return nil, m.errors[m.callCount]
	}
	if m.callCount < len(m.responses) {
		return m.responses[m.callCount], nil
	}
	return nil, io.EOF
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		envToken  string
		wantError bool
	}{
		{
			name:      "valid token",
			token:     "test-token",
			wantError: false,
		},
		{
			name:      "empty token with env",
			token:     "",
			envToken:  "env-token",
			wantError: false,
		},
		{
			name:      "empty token no env",
			token:     "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envToken != "" {
				t.Setenv("READWISE_TOKEN", tt.envToken)
			} else {
				t.Setenv("READWISE_TOKEN", "")
			}

			client, err := NewClient(tt.token)
			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got nil")
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

func TestVerifyToken(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantValid  bool
		wantError  bool
	}{
		{
			name:       "valid token",
			statusCode: http.StatusNoContent,
			wantValid:  true,
			wantError:  false,
		},
		{
			name:       "invalid token",
			statusCode: http.StatusUnauthorized,
			wantValid:  false,
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockHTTPClient{
				responses: []*http.Response{
					{
						StatusCode: tt.statusCode,
						Body:       io.NopCloser(bytes.NewReader([]byte{})),
					},
				},
			}

			client, _ := NewClient("test-token", WithHTTPClient(mock))
			valid, err := client.VerifyToken()

			if tt.wantError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if valid != tt.wantValid {
				t.Errorf("got valid=%v, want %v", valid, tt.wantValid)
			}
		})
	}
}

func TestGetInboxItems(t *testing.T) {
	now := FlexibleTime{Time: time.Now()}
	item1 := Item{
		ID:        "item1",
		Title:     "Test Article 1",
		URL:       "https://example.com/1",
		Category:  "article",
		SavedAt:   now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	item2 := Item{
		ID:        "item2",
		Title:     "Test Article 2",
		URL:       "https://example.com/2",
		Category:  "article",
		SavedAt:   now,
		CreatedAt: now,
		UpdatedAt: now,
	}

	response := ListResponse{
		Count:   2,
		Results: []Item{item1, item2},
	}

	body, _ := json.Marshal(response)

	mock := &mockHTTPClient{
		responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(body)),
			},
		},
	}

	client, _ := NewClient("test-token", WithHTTPClient(mock))
	items, err := client.GetInboxItems(FetchOptions{DaysAgo: 7})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
	if items[0].ID != "item1" {
		t.Errorf("expected item1, got %s", items[0].ID)
	}
}

func TestGetInboxItemsWithPagination(t *testing.T) {
	now := FlexibleTime{Time: time.Now()}
	cursor1 := "next-page-cursor"

	item1 := Item{
		ID:        "item1",
		Title:     "Test Article 1",
		URL:       "https://example.com/1",
		Category:  "article",
		SavedAt:   now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	item2 := Item{
		ID:        "item2",
		Title:     "Test Article 2",
		URL:       "https://example.com/2",
		Category:  "article",
		SavedAt:   now,
		CreatedAt: now,
		UpdatedAt: now,
	}

	response1 := ListResponse{
		Count:          1,
		NextPageCursor: &cursor1,
		Results:        []Item{item1},
	}
	response2 := ListResponse{
		Count:   1,
		Results: []Item{item2},
	}

	body1, _ := json.Marshal(response1)
	body2, _ := json.Marshal(response2)

	mock := &mockHTTPClient{
		responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(body1)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(body2)),
			},
		},
	}

	client, _ := NewClient("test-token", WithHTTPClient(mock))
	items, err := client.GetInboxItems(FetchOptions{DaysAgo: 7})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items from pagination, got %d", len(items))
	}
	if mock.callCount != 2 {
		t.Errorf("expected 2 API calls for pagination, got %d", mock.callCount)
	}
}

func TestExtractForPerplexity(t *testing.T) {
	now := FlexibleTime{Time: time.Now()}
	items := []Item{
		{
			ID:        "item1",
			Title:     "Test Article",
			URL:       "https://example.com",
			Category:  "article",
			SavedAt:   now,
			CreatedAt: now,
			UpdatedAt: now,
			Tags:      []string{"tag1", "tag2"},
		},
	}

	simplified := ExtractForPerplexity(items)

	if len(simplified) != 1 {
		t.Errorf("expected 1 simplified item, got %d", len(simplified))
	}
	if simplified[0].ID != "item1" {
		t.Errorf("expected ID item1, got %s", simplified[0].ID)
	}
	if len(simplified[0].Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(simplified[0].Tags))
	}
}

func TestFetchOptionsDefaults(t *testing.T) {
	opts := DefaultFetchOptions()

	if opts.DaysAgo != 7 {
		t.Errorf("expected DaysAgo=7, got %d", opts.DaysAgo)
	}
	if opts.Location != "new" {
		t.Errorf("expected Location='new', got %s", opts.Location)
	}
}
