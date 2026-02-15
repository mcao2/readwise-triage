package readwise

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	defaultBaseURL = "https://readwise.io/api/v3"
	authURL        = "https://readwise.io/api/v2/auth/"
	maxRetries     = 3
	retryDelay     = time.Second
)

// HTTPClient defines the interface for HTTP operations
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client handles communication with the Readwise API
type Client struct {
	token      string
	baseURL    string
	httpClient HTTPClient
}

// ClientOption allows configuring the Client
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient HTTPClient) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithBaseURL sets a custom base URL
func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = url
	}
}

// NewClient creates a new Readwise API client
func NewClient(token string, opts ...ClientOption) (*Client, error) {
	if token == "" {
		token = os.Getenv("READWISE_TOKEN")
	}
	if token == "" {
		return nil, fmt.Errorf("READWISE_TOKEN environment variable not set")
	}

	client := &Client{
		token:      token,
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}

	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

// VerifyToken checks if the API token is valid
func (c *Client) VerifyToken() (bool, error) {
	req, err := http.NewRequest("GET", authURL, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", "Token "+c.token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusNoContent, nil
}

// doRequest performs an HTTP request with retry logic
func (c *Client) doRequest(req *http.Request) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelay * time.Duration(attempt))
		}

		req.Header.Set("Authorization", "Token "+c.token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			retryAfter := resp.Header.Get("Retry-After")
			if seconds, err := strconv.Atoi(retryAfter); err == nil {
				time.Sleep(time.Duration(seconds) * time.Second)
			} else {
				time.Sleep(retryDelay * time.Duration(attempt+1))
			}
			lastErr = fmt.Errorf("rate limited: %d", resp.StatusCode)
			continue
		}

		if resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", maxRetries, lastErr)
}

// decodeJSON reads and decodes JSON from response body
func decodeJSON(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}
