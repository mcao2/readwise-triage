package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mcao2/readwise-triage/internal/config"
	"github.com/mcao2/readwise-triage/internal/ui"
)

func main() {
	// Load config; run interactive setup if first run
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	if config.NeedsSetup(cfg) {
		cfg, err = config.Setup(cfg)
		if err != nil {
			fmt.Printf("Error saving config: %v\n", err)
			os.Exit(1)
		}
	}

	// Initialize the UI model
	m := ui.NewModel()

	// Create the Bubble Tea program with alternate screen (clears terminal)
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),       // Use alternate screen buffer (clears terminal)
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
