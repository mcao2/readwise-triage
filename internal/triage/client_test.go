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
		name     string
		provider string
		apiKey   string
		opts     []LLMOption
		wantErr  bool
	}{
		{
			name:     "openai with key",
			provider: "openai",
			apiKey:   "sk-test",
			wantErr:  false,
		},
		{
			name:     "perplexity with key",
			provider: "perplexity",
			apiKey:   "pplx-test",
			wantErr:  false,
		},
		{
			name:     "ollama no key needed",
			provider: "ollama",
			apiKey:   "",
			wantErr:  false,
		},
		{
			name:     "empty provider defaults to openai",
			provider: "",
			apiKey:   "sk-test",
			wantErr:  false,
		},
		{
			name:     "openai without key fails",
			provider: "openai",
			apiKey:   "",
			wantErr:  true,
		},
		{
			name:     "unknown provider without base_url fails",
			provider: "custom",
			apiKey:   "key",
			wantErr:  true,
		},
		{
			name:     "unknown provider with base_url and model works",
			provider: "custom",
			apiKey:   "key",
			opts: []LLMOption{
				WithLLMBaseURL("http://localhost:8080/v1/chat/completions"),
				WithLLMModel("my-model"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear env vars

			client, err := NewLLMClient(tt.provider, tt.apiKey, tt.opts...)
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
	client, err := NewLLMClient("openai", "sk-test", WithLLMModel("gpt-4o"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.model != "gpt-4o" {
		t.Errorf("expected model 'gpt-4o', got %q", client.model)
	}

	client2, err := NewLLMClient("openai", "sk-test", WithLLMBaseURL("http://custom/v1/chat/completions"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client2.baseURL != "http://custom/v1/chat/completions" {
		t.Errorf("expected custom baseURL, got %q", client2.baseURL)
	}

	customHTTP := &http.Client{}
	client3, err := NewLLMClient("openai", "sk-test", WithLLMHTTPClient(customHTTP))
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

	client, _ := NewLLMClient("openai", "sk-test", WithLLMBaseURL(server.URL))
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

	client, _ := NewLLMClient("openai", "sk-test", WithLLMBaseURL(server.URL))
	_, err := client.TriageItems(`[{"id":"1","title":"Test"}]`)
	if err == nil {
		t.Error("expected error on 500 response")
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

	client, _ := NewLLMClient("openai", "sk-test", WithLLMBaseURL(server.URL))
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

func TestLLMClientNoAuthForOllama(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("expected no auth header for ollama, got %q", auth)
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

	client, err := NewLLMClient("ollama", "", WithLLMBaseURL(server.URL))
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
	// Verify the auto-triage prompt contains the placeholder and key instructions
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
	// Should NOT contain the heavy fields from the full export prompt
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
