package ui

import "github.com/charmbracelet/lipgloss"

// Styles holds all the UI styles
type Styles struct {
	Title     lipgloss.Style
	Normal    lipgloss.Style
	Help      lipgloss.Style
	Highlight lipgloss.Style
	Selected  lipgloss.Style
	Error     lipgloss.Style
	Success   lipgloss.Style
}

// DefaultStyles returns the default style set
func DefaultStyles() Styles {
	return Styles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			PaddingTop(1).
			PaddingBottom(1),

		Normal: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")),

		Help: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#737373")).
			Italic(true),

		Highlight: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#04B575")),

		Selected: lipgloss.NewStyle().
			Background(lipgloss.Color("#7D56F4")).
			Foreground(lipgloss.Color("#FFFFFF")),

		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")),

		Success: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")),
	}
}
