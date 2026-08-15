package triage

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewLLMClient(t *testing.T) {
	tests := []struct {
		name    string
		apiKey  string
		opts    []LLMOption
		wantErr bool
	}{
		{
			name:    "valid config with key",
			apiKey:  "sk-test",
			opts:    []LLMOption{WithLLMBaseURL("https://api.openai.com"), WithLLMModel("gpt-4o-mini")},
			wantErr: false,
		},
		{
			name:    "local provider no key needed",
			apiKey:  "",
			opts:    []LLMOption{WithLLMBaseURL("http://localhost:11434"), WithLLMModel("llama3")},
			wantErr: false,
		},
		{
			name:    "missing base_url fails",
			apiKey:  "sk-test",
			opts:    []LLMOption{WithLLMModel("gpt-4o")},
			wantErr: true,
		},
		{
			name:    "missing model fails",
			apiKey:  "sk-test",
			opts:    []LLMOption{WithLLMBaseURL("https://api.openai.com")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewLLMClient(tt.apiKey, tt.opts...)
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

func TestLLMClientOptions(t *testing.T) {
	client, err := NewLLMClient("sk-test", WithLLMModel("gpt-4o"), WithLLMBaseURL("https://api.openai.com"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.model != "gpt-4o" {
		t.Errorf("expected model 'gpt-4o', got %q", client.model)
	}

	client2, err := NewLLMClient("sk-test", WithLLMBaseURL("http://custom/v1/chat/completions"), WithLLMModel("test"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client2.baseURL != "http://custom/v1/chat/completions" {
		t.Errorf("expected custom baseURL, got %q", client2.baseURL)
	}

	customHTTP := &http.Client{}
	client3, err := NewLLMClient("sk-test", WithLLMHTTPClient(customHTTP), WithLLMBaseURL("https://api.openai.com"), WithLLMModel("test"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client3.httpClient != customHTTP {
		t.Error("expected custom HTTP client to be set")
	}
}

func TestLLMClientTriageItems(t *testing.T) {
	triageResult := []Result{
		{
			ID:    "item1",
			Title: "Test Article",
			URL:   "https://example.com",
			TriageDecision: TriageDecision{
				Action:   "read_now",
				Priority: "high",
				Reason:   "Important",
			},
		},
	}
	resultJSON, _ := json.Marshal(triageResult)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Errorf("expected Bearer auth, got %q", r.Header.Get("Authorization"))
		}
		resp := ChatResponse{
			Choices: []struct {
				Message ChatMessage `json:"message"`
			}{
				{Message: ChatMessage{Role: "assistant", Content: string(resultJSON)}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, _ := NewLLMClient("sk-test", WithLLMBaseURL(server.URL), WithLLMModel("test"))
	results, err := client.TriageItems(`[{"id":"item1","title":"Test"}]`)
	if err != nil {
		t.Fatalf("TriageItems failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if results[0].ID != "item1" {
		t.Errorf("expected id 'item1', got %q", results[0].ID)
	}
}

func TestLLMClientTriageItemsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client, _ := NewLLMClient("sk-test", WithLLMBaseURL(server.URL), WithLLMModel("test"))
	_, err := client.TriageItems(`[{"id":"1","title":"Test"}]`)
	if err == nil {
		t.Error("expected error on 500 response")
	}
}

func TestLLMClientTriageItems4xxNoRetry(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":{"message":"Invalid URL (POST /v1)","type":"invalid_request_error"}}`))
	}))
	defer server.Close()

	client, _ := NewLLMClient("sk-test", WithLLMBaseURL(server.URL), WithLLMModel("test"))
	_, err := client.TriageItems(`[{"id":"1","title":"Test"}]`)
	if err == nil {
		t.Error("expected error on 404 response")
	}
	if callCount != 1 {
		t.Errorf("expected 1 call (no retry for 4xx), got %d", callCount)
	}
	if !contains(err.Error(), "Invalid URL") {
		t.Errorf("expected parsed error message, got %q", err.Error())
	}
}

func TestLLMClientTriageItemsNonJSONResponse(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body>Not Found</body></html>"))
	}))
	defer server.Close()

	client, _ := NewLLMClient("sk-test", WithLLMBaseURL(server.URL), WithLLMModel("test"))
	_, err := client.TriageItems(`[{"id":"1","title":"Test"}]`)
	if err == nil {
		t.Error("expected error on HTML response")
	}
	if callCount != 1 {
		t.Errorf("expected 1 call (no retry for non-JSON), got %d", callCount)
	}
	if !contains(err.Error(), "not JSON") {
		t.Errorf("expected 'not JSON' in error, got %q", err.Error())
	}
}

func TestLLMClientTriageItemsRetry(t *testing.T) {
	callCount := 0
	triageResult := []Result{
		{
			ID:    "item1",
			Title: "Test",
			URL:   "https://example.com",
			TriageDecision: TriageDecision{
				Action:   "later",
				Priority: "low",
				Reason:   "Not urgent",
			},
		},
	}
	resultJSON, _ := json.Marshal(triageResult)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("error"))
			return
		}
		resp := ChatResponse{
			Choices: []struct {
				Message ChatMessage `json:"message"`
			}{
				{Message: ChatMessage{Role: "assistant", Content: string(resultJSON)}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, _ := NewLLMClient("sk-test", WithLLMBaseURL(server.URL), WithLLMModel("test"))
	results, err := client.TriageItems(`[{"id":"item1","title":"Test"}]`)
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls (1 retry), got %d", callCount)
	}
}

func TestLLMClientNoAuthWhenNoKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("expected no auth header when no key, got %q", auth)
		}
		triageResult := []Result{
			{
				ID:    "item1",
				Title: "Test",
				URL:   "https://example.com",
				TriageDecision: TriageDecision{
					Action:   "archive",
					Priority: "low",
					Reason:   "Not relevant",
				},
			},
		}
		resultJSON, _ := json.Marshal(triageResult)
		resp := ChatResponse{
			Choices: []struct {
				Message ChatMessage `json:"message"`
			}{
				{Message: ChatMessage{Role: "assistant", Content: string(resultJSON)}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := NewLLMClient("", WithLLMBaseURL(server.URL), WithLLMModel("test"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	results, err := client.TriageItems(`[{"id":"item1","title":"Test"}]`)
	if err != nil {
		t.Fatalf("TriageItems failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestAutoTriagePromptTemplate(t *testing.T) {
	if AutoTriagePromptTemplate == "" {
		t.Error("AutoTriagePromptTemplate is empty")
	}

	formatted := fmt.Sprintf(AutoTriagePromptTemplate, `[{"id":"1","title":"Test"}]`)
	if !contains(formatted, `"id":"1"`) {
		t.Error("expected items JSON to be interpolated into prompt")
	}
	if !contains(formatted, "Return ONLY a JSON array") {
		t.Error("expected JSON-only output instruction in auto prompt")
	}
	if contains(formatted, "credibility_check") {
		t.Error("auto prompt should not contain credibility_check")
	}
	if contains(formatted, "reading_guide") {
		t.Error("auto prompt should not contain reading_guide")
	}
	if contains(formatted, "content_analysis") {
		t.Error("auto prompt should not contain content_analysis")
	}
}

func TestLLMClientAutoAppendEndpoint(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		wantURL string
	}{
		{
			name:    "base url gets chat completions appended",
			baseURL: "https://api.openai.com",
			wantURL: "https://api.openai.com/v1/chat/completions",
		},
		{
			name:    "base url already has chat completions preserved",
			baseURL: "https://api.longcat.chat/openai/v1/chat/completions",
			wantURL: "https://api.longcat.chat/openai/v1/chat/completions",
		},
		{
			name:    "base url with trailing slash gets chat completions appended",
			baseURL: "https://api.openai.com/",
			wantURL: "https://api.openai.com/v1/chat/completions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewLLMClient("test-key", WithLLMBaseURL(tt.baseURL), WithLLMModel("test-model"))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if client.baseURL != tt.wantURL {
				t.Errorf("expected baseURL %q, got %q", tt.wantURL, client.baseURL)
			}
		})
	}
}
