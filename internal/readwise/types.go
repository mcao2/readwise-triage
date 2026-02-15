package readwise

import (
	"time"
)

// Item represents a Readwise Reader document
type Item struct {
	ID              string     `json:"id"`
	URL             string     `json:"source_url"` // Original article URL
	ReaderURL       string     `json:"url"`        // Readwise Reader URL
	Title           string     `json:"title"`
	Author          string     `json:"author"`
	Source          string     `json:"source"` // Source type (e.g., "Reader RSS")
	SiteName        string     `json:"site_name"`
	Category        string     `json:"category"` // article/rss/tweet/pdf/video
	WordCount       int        `json:"word_count"`
	ReadingTime     string     `json:"reading_time"`
	PublishedDate   *time.Time `json:"published_date"`
	SavedAt         time.Time  `json:"saved_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	Tags            []string   `json:"tags"`
	Summary         string     `json:"summary"`
	Notes           string     `json:"notes"`
	ReadingProgress float64    `json:"reading_progress"`
	FirstOpenedAt   *time.Time `json:"first_opened_at"`
	LastOpenedAt    *time.Time `json:"last_opened_at"`
}

// SimplifiedItem contains only the fields needed for LLM processing
type SimplifiedItem struct {
	ID              string     `json:"id"`
	URL             string     `json:"url"`
	ReaderURL       string     `json:"reader_url"`
	Title           string     `json:"title"`
	Author          string     `json:"author"`
	Source          string     `json:"source"`
	SiteName        string     `json:"site_name"`
	Category        string     `json:"category"`
	WordCount       int        `json:"word_count"`
	ReadingTime     string     `json:"reading_time"`
	PublishedDate   *time.Time `json:"published_date"`
	SavedAt         time.Time  `json:"saved_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	Tags            []string   `json:"tags"`
	Summary         string     `json:"summary"`
	Notes           string     `json:"notes"`
	ReadingProgress float64    `json:"reading_progress"`
	FirstOpenedAt   *time.Time `json:"first_opened_at"`
	LastOpenedAt    *time.Time `json:"last_opened_at"`
}

// ListResponse represents the API response structure
type ListResponse struct {
	Count          int     `json:"count"`
	NextPageCursor *string `json:"nextPageCursor"`
	Results        []Item  `json:"results"`
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
