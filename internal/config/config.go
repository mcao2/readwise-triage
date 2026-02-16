package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config holds application configuration
type Config struct {
	ReadwiseToken    string `yaml:"readwise_token"`
	PerplexityAPIKey string `yaml:"perplexity_api_key"`
	InboxDaysAgo     int    `yaml:"inbox_days_ago"`
	FeedDaysAgo      int    `yaml:"feed_days_ago"`
	Theme            string `yaml:"theme"`
	UseLLMTriage     bool   `yaml:"use_llm_triage"`
	Location         string `yaml:"location"`
}

// Load loads configuration from config file and environment variables
// Environment variables take precedence over config file values
func Load() (*Config, error) {
	cfg := &Config{
		InboxDaysAgo: 7,
		FeedDaysAgo:  7,
	}

	// Load from config file first
	if err := cfg.loadFromFile(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}

	// Environment variables override config file
	cfg.loadFromEnv()

	return cfg, nil
}

func (c *Config) loadFromFile() error {
	configPath := getConfigPath()
	if configPath == "" {
		return os.ErrNotExist
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, c); err != nil {
		return err
	}

	// Backward compat: if inbox_days_ago was not set, check legacy default_days_ago
	if c.InboxDaysAgo == 0 {
		var legacy struct {
			DefaultDaysAgo int `yaml:"default_days_ago"`
		}
		if err := yaml.Unmarshal(data, &legacy); err == nil && legacy.DefaultDaysAgo != 0 {
			c.InboxDaysAgo = legacy.DefaultDaysAgo
		}
	}

	return nil
}

func (c *Config) loadFromEnv() {
	if token := os.Getenv("READWISE_TOKEN"); token != "" {
		c.ReadwiseToken = token
	}
	if apiKey := os.Getenv("PERPLEXITY_API_KEY"); apiKey != "" {
		c.PerplexityAPIKey = apiKey
	}
	// Prefer INBOX_DAYS_AGO, fall back to legacy DEFAULT_DAYS_AGO
	if daysStr := os.Getenv("INBOX_DAYS_AGO"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil {
			c.InboxDaysAgo = d
		}
	} else if daysStr := os.Getenv("DEFAULT_DAYS_AGO"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil {
			c.InboxDaysAgo = d
		}
	}
}

// getConfigPath returns the path to the config file
// Priority: $READWISE_TRIAGE_CONFIG > ~/.config/readwise-triage/config.yaml
func getConfigPath() string {
	if configPath := os.Getenv("READWISE_TRIAGE_CONFIG"); configPath != "" {
		return configPath
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".config", "readwise-triage", "config.yaml")
}

func GetConfigDir() (string, error) {
	configPath := getConfigPath()
	if configPath == "" {
		return "", fmt.Errorf("cannot determine config path")
	}
	return filepath.Dir(configPath), nil
}

// EnsureConfigDir ensures the config directory exists
func EnsureConfigDir() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return configDir, nil
}

// SaveExampleConfig creates an example config file
func SaveExampleConfig() error {
	configDir, err := EnsureConfigDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "config.yaml")

	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil {
		return nil // Already exists, don't overwrite
	}

	example := `# Readwise Triage Configuration
# Get your token at: https://readwise.io/access_token

# Required: Your Readwise API token
readwise_token: "your_token_here"

# Optional: Perplexity API key for LLM auto-triage
# Get your key at: https://www.perplexity.ai/settings/api
perplexity_api_key: "your_api_key_here"

# Optional: Default number of days to fetch for inbox (default: 7)
inbox_days_ago: 7

# Optional: Default number of days to fetch for feed (default: 7)
feed_days_ago: 7

# Optional: Color theme (default, catppuccin, dracula, nord, gruvbox)
theme: "default"

# Optional: Use LLM auto-triage by default (default: true)
use_llm_triage: true
`

	return os.WriteFile(configPath, []byte(example), 0600)
}

func (c *Config) Save() error {
	configDir, err := EnsureConfigDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "config.yaml")

	// Load existing config to preserve fields like tokens
	existing := &Config{InboxDaysAgo: 7, FeedDaysAgo: 7}
	if data, err := os.ReadFile(configPath); err == nil {
		yaml.Unmarshal(data, existing)
	}

	// Update only the fields we manage (not tokens from env vars)
	existing.InboxDaysAgo = c.InboxDaysAgo
	existing.FeedDaysAgo = c.FeedDaysAgo
	existing.Theme = c.Theme
	existing.UseLLMTriage = c.UseLLMTriage
	existing.Location = c.Location
	// Note: We preserve existing.ReadwiseToken and existing.PerplexityAPIKey

	data, err := yaml.Marshal(existing)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	header := []byte("# Readwise Triage Configuration\n# Note: Sensitive values (tokens) can be set via environment variables or this file\n\n")
	return os.WriteFile(configPath, append(header, data...), 0600)
}
