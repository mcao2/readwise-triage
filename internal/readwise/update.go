package readwise

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// UpdateRequest represents a single document update
type UpdateRequest struct {
	DocumentID string   `json:"document_id"`
	Location   string   `json:"location,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Notes      string   `json:"notes,omitempty"`
}

// BatchUpdateResult tracks the result of batch updates
type BatchUpdateResult struct {
	Total   int
	Success int
	Failed  int
	Errors  []error
}

// UpdateDocument updates a single document
func (c *Client) UpdateDocument(update UpdateRequest) error {
	payload := map[string]interface{}{
		"document_id": update.DocumentID,
	}

	if update.Location != "" {
		payload["location"] = update.Location
	}
	if len(update.Tags) > 0 {
		payload["tags"] = update.Tags
	}
	if update.Notes != "" {
		payload["notes"] = update.Notes
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal update: %w", err)
	}

	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/update/%s/", c.baseURL, update.DocumentID), bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.doRequest(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("update failed with status %d", resp.StatusCode)
	}

	return nil
}

// BatchUpdate updates multiple documents with rate limiting
func (c *Client) BatchUpdate(updates []UpdateRequest, progressChan chan<- BatchUpdateProgress) (*BatchUpdateResult, error) {
	result := &BatchUpdateResult{
		Total:  len(updates),
		Errors: make([]error, 0),
	}

	rateLimiter := time.NewTicker(2 * time.Second)
	defer rateLimiter.Stop()

	for i, update := range updates {
		<-rateLimiter.C

		err := c.UpdateDocument(update)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Errorf("update %s: %w", update.DocumentID, err))
		} else {
			result.Success++
		}

		if progressChan != nil {
			progressChan <- BatchUpdateProgress{
				Current: i + 1,
				Total:   len(updates),
				ItemID:  update.DocumentID,
				Success: err == nil,
			}
		}
	}

	return result, nil
}

// BatchUpdateProgress tracks progress of batch updates
type BatchUpdateProgress struct {
	Current int
	Total   int
	ItemID  string
	Success bool
}
