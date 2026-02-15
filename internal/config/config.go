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
	DefaultDaysAgo   int    `yaml:"default_days_ago"`
	Theme            string `yaml:"theme"`
}

// Load loads configuration from config file and environment variables
// Environment variables take precedence over config file values
func Load() (*Config, error) {
	cfg := &Config{
		DefaultDaysAgo: 7,
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

	return yaml.Unmarshal(data, c)
}

func (c *Config) loadFromEnv() {
	if token := os.Getenv("READWISE_TOKEN"); token != "" {
		c.ReadwiseToken = token
	}
	if apiKey := os.Getenv("PERPLEXITY_API_KEY"); apiKey != "" {
		c.PerplexityAPIKey = apiKey
	}
	if daysStr := os.Getenv("DEFAULT_DAYS_AGO"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil {
			c.DefaultDaysAgo = d
		}
	}
}

// getConfigPath returns the path to the config file
// Priority: $READWISE_TRIAGE_CONFIG > ~/.config/readwise-triage/config.yaml
func getConfigPath() string {
	// Check environment variable override
	if configPath := os.Getenv("READWISE_TRIAGE_CONFIG"); configPath != "" {
		return configPath
	}

	// Default location
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".config", "readwise-triage", "config.yaml")
}

// EnsureConfigDir ensures the config directory exists
func EnsureConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(home, ".config", "readwise-triage")
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

# Optional: Default number of days to fetch (default: 7)
default_days_ago: 7

# Optional: Color theme (default, catppuccin, dracula, nord, gruvbox)
theme: "default"
`

	return os.WriteFile(configPath, []byte(example), 0600)
}

// Save writes the configuration to the config file
// Only saves non-sensitive settings (theme, mode preferences)
// Does NOT save tokens (those should be in env vars)
func (c *Config) Save() error {
	configDir, err := EnsureConfigDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "config.yaml")

	// Create config struct with only non-sensitive data
	safeConfig := &Config{
		DefaultDaysAgo: c.DefaultDaysAgo,
		Theme:          c.Theme,
	}

	data, err := yaml.Marshal(safeConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	header := []byte("# Readwise Triage Configuration\n# Note: Sensitive values (tokens) should be set via environment variables\n\n")
	return os.WriteFile(configPath, append(header, data...), 0600)
}
