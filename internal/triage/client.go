package triage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultLLMTimeout = 120 * time.Second
	defaultMaxRetries = 3
	defaultRetryDelay = time.Second
)

// Provider presets for known LLM providers
var providerDefaults = map[string]struct {
	BaseURL   string
	Model     string
	APIFormat string
}{
	"perplexity": {BaseURL: "https://api.perplexity.ai/chat/completions", Model: "sonar", APIFormat: "openai"},
	"openai":     {BaseURL: "https://api.openai.com/v1/chat/completions", Model: "gpt-4o-mini", APIFormat: "openai"},
	"anthropic":  {BaseURL: "https://api.anthropic.com/v1/messages", Model: "claude-sonnet-4-5-20250929", APIFormat: "anthropic"},
	"ollama":     {BaseURL: "http://localhost:11434/v1/chat/completions", Model: "llama3", APIFormat: "openai"},
}

// ChatMessage represents a message in the chat API
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents the API request body
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

// ChatResponse represents the API response
type ChatResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// AnthropicRequest represents the Anthropic /v1/messages request body
type AnthropicRequest struct {
	Model     string        `json:"model"`
	MaxTokens int           `json:"max_tokens"`
	System    string        `json:"system,omitempty"`
	Messages  []ChatMessage `json:"messages"`
}

// AnthropicResponse represents the Anthropic /v1/messages response
type AnthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// LLMClient handles communication with any OpenAI-compatible chat completions API
type LLMClient struct {
	provider   string
	apiFormat  string // "openai" (default) or "anthropic"
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

// WithLLMAPIFormat sets the wire format ("openai" or "anthropic")
func WithLLMAPIFormat(format string) LLMOption {
	return func(c *LLMClient) {
		if format != "" {
			c.apiFormat = format
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
		provider:   provider,
		apiFormat:  defaults.APIFormat,
		apiKey:     apiKey,
		model:      defaults.Model,
		baseURL:    defaults.BaseURL,
		httpClient: &http.Client{Timeout: defaultLLMTimeout},
	}

	for _, opt := range opts {
		opt(client)
	}

	// Default api_format to "openai" if not set
	if client.apiFormat == "" {
		client.apiFormat = "openai"
	}

	// Auto-append standard path if base URL has no path component
	if !strings.Contains(strings.TrimPrefix(strings.TrimPrefix(client.baseURL, "https://"), "http://"), "/") {
		switch client.apiFormat {
		case "anthropic":
			client.baseURL = strings.TrimRight(client.baseURL, "/") + "/v1/messages"
		default:
			client.baseURL = strings.TrimRight(client.baseURL, "/") + "/v1/chat/completions"
		}
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

	var body []byte
	var err error

	if c.apiFormat == "anthropic" {
		reqBody := AnthropicRequest{
			Model:     c.model,
			MaxTokens: 4096,
			System:    "You are a helpful assistant that analyzes reading materials and provides structured triage recommendations. Return ONLY valid JSON.",
			Messages: []ChatMessage{
				{Role: "user", Content: prompt},
			},
		}
		body, err = json.Marshal(reqBody)
	} else {
		reqBody := ChatRequest{
			Model: c.model,
			Messages: []ChatMessage{
				{Role: "system", Content: "You are a helpful assistant that analyzes reading materials and provides structured triage recommendations. Return ONLY valid JSON."},
				{Role: "user", Content: prompt},
			},
		}
		body, err = json.Marshal(reqBody)
	}
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
			// Don't retry client errors (4xx)
			var noRetry *errNoRetry
			if errors.As(err, &noRetry) {
				return nil, noRetry.err
			}
			lastErr = err
			continue
		}
		return results, nil
	}

	return nil, fmt.Errorf("triage failed after %d retries: %w", defaultMaxRetries, lastErr)
}

// errNoRetry wraps errors that should not be retried (e.g., 4xx client errors).
type errNoRetry struct {
	err error
}

func (e *errNoRetry) Error() string { return e.err.Error() }
func (e *errNoRetry) Unwrap() error { return e.err }

func (c *LLMClient) doRequest(body []byte) ([]Result, error) {
	req, err := http.NewRequest("POST", c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	if c.apiFormat == "anthropic" {
		req.Header.Set("x-api-key", c.apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	} else if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		apiErr := parseAPIError(resp.StatusCode, respBody)
		// Don't retry client errors (4xx) — only server errors are transient
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return nil, &errNoRetry{err: apiErr}
		}
		return nil, apiErr
	}

	content, err := c.extractContent(respBody)
	if err != nil {
		return nil, err
	}

	results, err := ParseTriageResponse(content)
	if err != nil {
		return nil, &errNoRetry{err: err}
	}
	return results, nil
}

// extractContent parses the response body and returns the text content,
// handling both OpenAI and Anthropic response formats.
func (c *LLMClient) extractContent(respBody []byte) (string, error) {
	if c.apiFormat == "anthropic" {
		var anthropicResp AnthropicResponse
		if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
			preview := string(respBody)
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			return "", &errNoRetry{err: fmt.Errorf("unexpected response (not JSON): %s", preview)}
		}
		if anthropicResp.Error != nil {
			return "", fmt.Errorf("API error: %s", anthropicResp.Error.Message)
		}
		for _, block := range anthropicResp.Content {
			if block.Type == "text" {
				return block.Text, nil
			}
		}
		return "", fmt.Errorf("no text content in Anthropic response")
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		preview := string(respBody)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return "", &errNoRetry{err: fmt.Errorf("unexpected response (not JSON): %s", preview)}
	}
	if chatResp.Error != nil {
		return "", fmt.Errorf("API error: %s", chatResp.Error.Message)
	}
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	return chatResp.Choices[0].Message.Content, nil
}

// parseAPIError extracts a human-readable message from an API error response.
// If the body is JSON with an error.message field, it uses that; otherwise falls back to raw body.
func parseAPIError(statusCode int, body []byte) error {
	var parsed struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &parsed) == nil && parsed.Error.Message != "" {
		return fmt.Errorf("API error (status %d): %s", statusCode, parsed.Error.Message)
	}
	return fmt.Errorf("API error (status %d): %s", statusCode, string(body))
}
