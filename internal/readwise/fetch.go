package readwise

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// FetchOptions contains optional parameters for fetching items
type FetchOptions struct {
	DaysAgo  int
	Location string
}

// DefaultFetchOptions returns default fetch options
func DefaultFetchOptions() FetchOptions {
	return FetchOptions{
		DaysAgo:  7,
		Location: "new", // inbox
	}
}

// GetInboxItems fetches inbox items from Readwise with pagination
func (c *Client) GetInboxItems(opts FetchOptions) ([]Item, error) {
	if opts.DaysAgo == 0 {
		opts.DaysAgo = DefaultFetchOptions().DaysAgo
	}
	if opts.Location == "" {
		opts.Location = DefaultFetchOptions().Location
	}

	startDate := time.Now().AddDate(0, 0, -opts.DaysAgo)
	updatedAfter := startDate.Format(time.RFC3339)

	var allItems []Item
	var cursor *string

	for {
		items, nextCursor, err := c.fetchPage(updatedAfter, opts.Location, cursor)
		if err != nil {
			return nil, err
		}

		allItems = append(allItems, items...)
		cursor = nextCursor

		if cursor == nil {
			break
		}
	}

	return allItems, nil
}

// fetchPage fetches a single page of results
func (c *Client) fetchPage(updatedAfter, location string, cursor *string) ([]Item, *string, error) {
	params := url.Values{}
	params.Set("location", location)
	params.Set("updatedAfter", updatedAfter)
	if cursor != nil {
		params.Set("pageCursor", *cursor)
	}

	reqURL := fmt.Sprintf("%s/list/?%s", c.baseURL, params.Encode())
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, nil, err
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("API request failed: %d", resp.StatusCode)
	}

	var result ListResponse
	if err := decodeJSON(resp.Body, &result); err != nil {
		return nil, nil, err
	}

	return result.Results, result.NextPageCursor, nil
}

// ExtractForPerplexity converts full items to simplified format for LLM processing
func ExtractForPerplexity(items []Item) []SimplifiedItem {
	simplified := make([]SimplifiedItem, 0, len(items))
	for _, item := range items {
		simplified = append(simplified, item.ToSimplified())
	}
	return simplified
}
