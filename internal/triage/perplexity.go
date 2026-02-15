package triage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	perplexityAPIURL = "https://api.perplexity.ai/chat/completions"
	defaultModel     = "sonar"
	maxRetries       = 3
	retryDelay       = time.Second
)

// PerplexityClient handles communication with Perplexity API
type PerplexityClient struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

// PerplexityOption allows configuring the client
type PerplexityOption func(*PerplexityClient)

// WithPerplexityHTTPClient sets a custom HTTP client
func WithPerplexityHTTPClient(client *http.Client) PerplexityOption {
	return func(c *PerplexityClient) {
		c.httpClient = client
	}
}

// WithModel sets a custom model
func WithModel(model string) PerplexityOption {
	return func(c *PerplexityClient) {
		c.model = model
	}
}

// WithPerplexityBaseURL sets a custom base URL (for testing)
func WithPerplexityBaseURL(url string) PerplexityOption {
	return func(c *PerplexityClient) {
		c.baseURL = url
	}
}

// NewPerplexityClient creates a new Perplexity API client
func NewPerplexityClient(apiKey string, opts ...PerplexityOption) (*PerplexityClient, error) {
	if apiKey == "" {
		apiKey = os.Getenv("PERPLEXITY_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("PERPLEXITY_API_KEY environment variable not set")
	}

	client := &PerplexityClient{
		apiKey:     apiKey,
		model:      defaultModel,
		baseURL:    perplexityAPIURL,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}

	for _, opt := range opts {
		opt(client)
	}

	return client, nil
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

// TriageItems sends items to Perplexity for triage and returns the results
func (c *PerplexityClient) TriageItems(itemsJSON string) ([]Result, error) {
	prompt := fmt.Sprintf(PromptTemplate, itemsJSON)

	reqBody := ChatRequest{
		Model: c.model,
		Messages: []ChatMessage{
			{Role: "system", Content: "You are a helpful assistant that analyzes reading materials and provides structured triage recommendations."},
			{Role: "user", Content: prompt},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelay * time.Duration(attempt))
		}

		results, err := c.doRequest(body)
		if err != nil {
			lastErr = err
			continue
		}
		return results, nil
	}

	return nil, fmt.Errorf("triage failed after %d retries: %w", maxRetries, lastErr)
}

func (c *PerplexityClient) doRequest(body []byte) ([]Result, error) {
	req, err := http.NewRequest("POST", c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
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
