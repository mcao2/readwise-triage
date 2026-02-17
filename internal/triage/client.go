package triage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultLLMTimeout = 120 * time.Second
	defaultMaxRetries = 3
	defaultRetryDelay = time.Second
)

// Provider presets for known LLM providers
var providerDefaults = map[string]struct {
	BaseURL string
	Model   string
}{
	"perplexity": {BaseURL: "https://api.perplexity.ai/chat/completions", Model: "sonar"},
	"openai":     {BaseURL: "https://api.openai.com/v1/chat/completions", Model: "gpt-4o-mini"},
	"ollama":     {BaseURL: "http://localhost:11434/v1/chat/completions", Model: "llama3"},
}

// LLMClient handles communication with any OpenAI-compatible chat completions API
type LLMClient struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

// LLMOption allows configuring the client
type LLMOption func(*LLMClient)

// WithLLMHTTPClient sets a custom HTTP client
func WithLLMHTTPClient(client *http.Client) LLMOption {
	return func(c *LLMClient) {
		c.httpClient = client
	}
}

// WithLLMModel sets a custom model
func WithLLMModel(model string) LLMOption {
	return func(c *LLMClient) {
		if model != "" {
			c.model = model
		}
	}
}

// WithLLMBaseURL sets a custom base URL
func WithLLMBaseURL(url string) LLMOption {
	return func(c *LLMClient) {
		if url != "" {
			c.baseURL = url
		}
	}
}

// NewLLMClient creates a new LLM API client.
// provider can be "perplexity", "openai", "ollama", or empty (defaults to openai).
// apiKey can be empty for providers that don't require it (e.g., ollama).
func NewLLMClient(provider, apiKey string, opts ...LLMOption) (*LLMClient, error) {
	if provider == "" {
		provider = "openai"
	}

	defaults, known := providerDefaults[provider]
	if !known {
		// Unknown provider: require explicit base_url via options
		defaults.BaseURL = ""
		defaults.Model = ""
	}

	client := &LLMClient{
		apiKey:     apiKey,
		model:      defaults.Model,
		baseURL:    defaults.BaseURL,
		httpClient: &http.Client{Timeout: defaultLLMTimeout},
	}

	for _, opt := range opts {
		opt(client)
	}

	// Validate: need a base URL
	if client.baseURL == "" {
		return nil, fmt.Errorf("LLM base_url is required for provider %q", provider)
	}

	// Validate: need a model
	if client.model == "" {
		return nil, fmt.Errorf("LLM model is required for provider %q", provider)
	}

	// API key is required for non-local providers
	if client.apiKey == "" && provider != "ollama" {
		return nil, fmt.Errorf("LLM api_key is required for provider %q", provider)
	}

	return client, nil
}

// TriageItems sends items to the LLM for triage and returns the results.
// It uses the lean auto-triage prompt that only requests fields consumed downstream.
func (c *LLMClient) TriageItems(itemsJSON string) ([]Result, error) {
	prompt := fmt.Sprintf(AutoTriagePromptTemplate, itemsJSON)

	reqBody := ChatRequest{
		Model: c.model,
		Messages: []ChatMessage{
			{Role: "system", Content: "You are a helpful assistant that analyzes reading materials and provides structured triage recommendations. Return ONLY valid JSON."},
			{Role: "user", Content: prompt},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < defaultMaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(defaultRetryDelay * time.Duration(attempt))
		}

		results, err := c.doRequest(body)
		if err != nil {
			lastErr = err
			continue
		}
		return results, nil
	}

	return nil, fmt.Errorf("triage failed after %d retries: %w", defaultMaxRetries, lastErr)
}

func (c *LLMClient) doRequest(body []byte) ([]Result, error) {
	req, err := http.NewRequest("POST", c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if chatResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	content := chatResp.Choices[0].Message.Content
	return ParseTriageResponse(content)
}
