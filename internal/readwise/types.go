package readwise

import (
	"fmt"
	"time"
)

// FlexibleTime is a time.Time that can parse multiple date formats
type FlexibleTime struct {
	time.Time
}

// UnmarshalJSON implements custom JSON unmarshaling for FlexibleTime
func (ft *FlexibleTime) UnmarshalJSON(data []byte) error {
	// Remove quotes from JSON string
	str := string(data)
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}

	// Try parsing with different formats
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, str); err == nil {
			ft.Time = t
			return nil
		}
	}

	return fmt.Errorf("unable to parse time: %s", str)
}

// MarshalJSON implements custom JSON marshaling
func (ft FlexibleTime) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", ft.Format(time.RFC3339))), nil
}

// Item represents a Readwise Reader document
type Item struct {
	ID              string        `json:"id"`
	URL             string        `json:"source_url"`
	ReaderURL       string        `json:"url"`
	Title           string        `json:"title"`
	Author          string        `json:"author"`
	Source          string        `json:"source"`
	SiteName        string        `json:"site_name"`
	Category        string        `json:"category"`
	WordCount       int           `json:"word_count"`
	ReadingTime     string        `json:"reading_time"`
	PublishedDate   *FlexibleTime `json:"published_date,omitempty"`
	SavedAt         FlexibleTime  `json:"saved_at"`
	CreatedAt       FlexibleTime  `json:"created_at"`
	UpdatedAt       FlexibleTime  `json:"updated_at"`
	Tags            []string      `json:"tags"`
	Summary         string        `json:"summary"`
	Notes           string        `json:"notes"`
	ReadingProgress float64       `json:"reading_progress"`
	FirstOpenedAt   *FlexibleTime `json:"first_opened_at,omitempty"`
	LastOpenedAt    *FlexibleTime `json:"last_opened_at,omitempty"`
}

// SimplifiedItem contains only the fields needed for LLM processing
type SimplifiedItem struct {
	ID              string        `json:"id"`
	URL             string        `json:"url"`
	ReaderURL       string        `json:"reader_url"`
	Title           string        `json:"title"`
	Author          string        `json:"author"`
	Source          string        `json:"source"`
	SiteName        string        `json:"site_name"`
	Category        string        `json:"category"`
	WordCount       int           `json:"word_count"`
	ReadingTime     string        `json:"reading_time"`
	PublishedDate   *FlexibleTime `json:"published_date,omitempty"`
	SavedAt         FlexibleTime  `json:"saved_at"`
	CreatedAt       FlexibleTime  `json:"created_at"`
	UpdatedAt       FlexibleTime  `json:"updated_at"`
	Tags            []string      `json:"tags"`
	Summary         string        `json:"summary"`
	Notes           string        `json:"notes"`
	ReadingProgress float64       `json:"reading_progress"`
	FirstOpenedAt   *FlexibleTime `json:"first_opened_at,omitempty"`
	LastOpenedAt    *FlexibleTime `json:"last_opened_at,omitempty"`
}

// ToSimplified converts a full Item to SimplifiedItem for LLM processing
func (i Item) ToSimplified() SimplifiedItem {
	return SimplifiedItem{
		ID:              i.ID,
		URL:             i.URL,
		ReaderURL:       i.ReaderURL,
		Title:           i.Title,
		Author:          i.Author,
		Source:          i.Source,
		SiteName:        i.SiteName,
		Category:        i.Category,
		WordCount:       i.WordCount,
		ReadingTime:     i.ReadingTime,
		PublishedDate:   i.PublishedDate,
		SavedAt:         i.SavedAt,
		CreatedAt:       i.CreatedAt,
		UpdatedAt:       i.UpdatedAt,
		Tags:            i.Tags,
		Summary:         i.Summary,
		Notes:           i.Notes,
		ReadingProgress: i.ReadingProgress,
		FirstOpenedAt:   i.FirstOpenedAt,
		LastOpenedAt:    i.LastOpenedAt,
	}
}

// ListResponse represents the API response structure
type ListResponse struct {
	Count          int     `json:"count"`
	NextPageCursor *string `json:"nextPageCursor"`
	Results        []Item  `json:"results"`
}
