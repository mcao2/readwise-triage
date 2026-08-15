package triage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultLLMTimeout = 120 * time.Second
	defaultMaxRetries = 3
	defaultRetryDelay = time.Second
)

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

// NewLLMClient creates a new LLM API client for any OpenAI-compatible endpoint.
// apiKey can be empty for local providers (e.g., ollama).
func NewLLMClient(apiKey string, opts ...LLMOption) (*LLMClient, error) {
	client := &LLMClient{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: defaultLLMTimeout},
	}

	for _, opt := range opts {
		opt(client)
	}

	if client.baseURL == "" {
		return nil, fmt.Errorf("LLM base_url is required")
	}

	// Auto-append standard API endpoint path if not already present
	baseURL := strings.TrimRight(client.baseURL, "/")
	switch {
	case strings.HasSuffix(baseURL, "/chat/completions"):
		// already complete, do nothing
	case strings.HasSuffix(baseURL, "/v1"):
		client.baseURL = baseURL + "/chat/completions"
	default:
		client.baseURL = baseURL + "/v1/chat/completions"
	}

	if client.model == "" {
		return nil, fmt.Errorf("LLM model is required")
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

	if c.apiKey != "" {
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

	// Debug: dump raw LLM response to stderr if READWISE_TRIAGE_DEBUG=1
	if os.Getenv("READWISE_TRIAGE_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[debug] LLM response (%d bytes): %s\n", len(content), truncate(content, 1000))
	}

	results, err := ParseTriageResponse(content)
	if err != nil {
		return nil, &errNoRetry{err: err}
	}
	return results, nil
}

// extractContent parses the OpenAI-compatible response body and returns the text content.
func (c *LLMClient) extractContent(respBody []byte) (string, error) {
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
