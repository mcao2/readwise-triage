package config

import (
	"os"
	"strconv"
)

// Config holds application configuration
type Config struct {
	ReadwiseToken    string
	PerplexityAPIKey string
	DefaultDaysAgo   int
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	daysAgo := 7
	if daysStr := os.Getenv("DEFAULT_DAYS_AGO"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil {
			daysAgo = d
		}
	}

	return &Config{
		ReadwiseToken:    os.Getenv("READWISE_TOKEN"),
		PerplexityAPIKey: os.Getenv("PERPLEXITY_API_KEY"),
		DefaultDaysAgo:   daysAgo,
	}, nil
}
