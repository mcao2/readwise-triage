package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// NeedsSetup returns true if required config is missing.
func NeedsSetup(cfg *Config) bool {
	return cfg.ReadwiseToken == ""
}

// Setup runs an interactive first-run configuration prompt.
// It asks for Readwise token (required) and LLM settings (optional),
// then saves the config to disk.
func Setup(cfg *Config) (*Config, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("╭─────────────────────────────────────╮")
	fmt.Println("│  Readwise Triage — First Run Setup   │")
	fmt.Println("╰─────────────────────────────────────╯")
	fmt.Println()

	// 1. Readwise token (required)
	cfg.ReadwiseToken = prompt(reader, "Readwise token", "", true)
	fmt.Println("  Get yours at: https://readwise.io/access_token")
	fmt.Println()

	// 2. LLM base URL (optional, default OpenAI)
	cfg.LLM.BaseURL = prompt(reader, "LLM base URL", "https://api.openai.com", false)
	fmt.Println("  Any OpenAI-compatible API (OpenAI, Ollama, OpenRouter, etc.)")
	fmt.Println()

	// 3. LLM API key (optional for local, required for cloud)
	cfg.LLM.APIKey = prompt(reader, "LLM API key", "", false)
	fmt.Println("  Leave empty for local providers (e.g., Ollama)")
	fmt.Println()

	// 4. LLM model (optional)
	cfg.LLM.Model = prompt(reader, "LLM model", "gpt-4o-mini", false)
	fmt.Println()

	// 5. Inbox days (optional, default 7)
	if cfg.InboxDaysAgo == 0 {
		cfg.InboxDaysAgo = 7
	}

	if err := cfg.Save(); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println("✓ Config saved to ~/.config/readwise-triage/config.yaml")
	fmt.Println()

	return cfg, nil
}

// prompt reads a line from the reader. If required and empty, retries.
// If optional and empty, returns the default.
func prompt(reader *bufio.Reader, label, defaultVal string, required bool) string {
	for {
		if defaultVal != "" {
			fmt.Printf("  %s [%s]: ", label, defaultVal)
		} else {
			fmt.Printf("  %s: ", label)
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			os.Exit(1)
		}

		val := strings.TrimSpace(line)
		if val == "" {
			if defaultVal != "" {
				return defaultVal
			}
			if !required {
				return ""
			}
			fmt.Println("  ⚠ This field is required. Please try again.")
			continue
		}
		return val
	}
}
