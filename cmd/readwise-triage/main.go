package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mcao2/readwise-triage/internal/ui"
)

func main() {
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
