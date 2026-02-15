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
	requests  []*http.Request
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	// Capture a copy of the request body so tests can inspect it
	if req.Body != nil {
		bodyBytes, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		clone := req.Clone(req.Context())
		clone.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		m.requests = append(m.requests, clone)
	} else {
		m.requests = append(m.requests, req)
	}
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
			Tags:      FlexibleTags{"tag1", "tag2"},
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

func TestDoRequest429WithRetryAfterHeader(t *testing.T) {
	mock := &mockHTTPClient{
		responses: []*http.Response{
			{
				StatusCode: http.StatusTooManyRequests,
				Header:     http.Header{"Retry-After": []string{"1"}},
				Body:       io.NopCloser(bytes.NewReader([]byte{})),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"ok":true}`))),
			},
		},
	}

	client, _ := NewClient("test-token", WithHTTPClient(mock))
	req, _ := http.NewRequest("GET", client.baseURL+"/list/", nil)

	resp, err := client.doRequest(req)
	if err != nil {
		t.Fatalf("expected success after retry, got error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if mock.callCount != 2 {
		t.Errorf("expected 2 calls (1 retry), got %d", mock.callCount)
	}
}

func TestDoRequest429FallsBackToExponentialBackoff(t *testing.T) {
	mock := &mockHTTPClient{
		responses: []*http.Response{
			{
				StatusCode: http.StatusTooManyRequests,
				Header:     http.Header{}, // no Retry-After
				Body:       io.NopCloser(bytes.NewReader([]byte{})),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"ok":true}`))),
			},
		},
	}

	client, _ := NewClient("test-token", WithHTTPClient(mock))
	req, _ := http.NewRequest("GET", client.baseURL+"/list/", nil)

	start := time.Now()
	resp, err := client.doRequest(req)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected success after retry, got error: %v", err)
	}
	defer resp.Body.Close()

	if mock.callCount != 2 {
		t.Errorf("expected 2 calls, got %d", mock.callCount)
	}
	// First attempt (attempt=0) with no Retry-After sleeps retryDelay * (0+1) = 1s
	if elapsed < 900*time.Millisecond {
		t.Errorf("expected backoff delay of ~1s, got %v", elapsed)
	}
}

func TestUpdateDocumentNoDocumentIDInBody(t *testing.T) {
	mock := &mockHTTPClient{
		responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{}`))),
			},
		},
	}

	client, _ := NewClient("test-token", WithHTTPClient(mock), WithBaseURL("http://fake"))

	err := client.UpdateDocument(UpdateRequest{
		DocumentID: "doc123",
		Location:   "archive",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.requests) == 0 {
		t.Fatal("expected at least one request")
	}

	body, _ := io.ReadAll(mock.requests[0].Body)
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to unmarshal request body: %v", err)
	}

	if _, exists := payload["document_id"]; exists {
		t.Error("document_id should not be in PATCH body (it's in the URL path)")
	}
	if payload["location"] != "archive" {
		t.Errorf("expected location=archive, got %v", payload["location"])
	}
}
