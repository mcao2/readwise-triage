package ui

import "github.com/charmbracelet/lipgloss"

// Theme represents a color theme for the application
type Theme struct {
	Name       string
	Primary    string
	Secondary  string
	Text       string
	Help       string
	Highlight  string
	Success    string
	Error      string
	Background string
}

// Available themes
var Themes = map[string]Theme{
	"default": {
		Name:       "Default",
		Primary:    "#7D56F4",
		Secondary:  "#FAFAFA",
		Text:       "#FAFAFA",
		Help:       "#737373",
		Highlight:  "#04B575",
		Success:    "#04B575",
		Error:      "#FF0000",
		Background: "#000000",
	},
	"catppuccin": {
		Name:       "Catppuccin",
		Primary:    "#89B4FA",
		Secondary:  "#CDD6F4",
		Text:       "#CDD6F4",
		Help:       "#6C7086",
		Highlight:  "#A6E3A1",
		Success:    "#A6E3A1",
		Error:      "#F38BA8",
		Background: "#1E1E2E",
	},
	"dracula": {
		Name:       "Dracula",
		Primary:    "#BD93F9",
		Secondary:  "#F8F8F2",
		Text:       "#F8F8F2",
		Help:       "#6272A4",
		Highlight:  "#50FA7B",
		Success:    "#50FA7B",
		Error:      "#FF5555",
		Background: "#282A36",
	},
	"nord": {
		Name:       "Nord",
		Primary:    "#88C0D0",
		Secondary:  "#D8DEE9",
		Text:       "#D8DEE9",
		Help:       "#4C566A",
		Highlight:  "#A3BE8C",
		Success:    "#A3BE8C",
		Error:      "#BF616A",
		Background: "#2E3440",
	},
	"gruvbox": {
		Name:       "Gruvbox",
		Primary:    "#D79921",
		Secondary:  "#EBDBB2",
		Text:       "#EBDBB2",
		Help:       "#928374",
		Highlight:  "#98971A",
		Success:    "#98971A",
		Error:      "#CC241D",
		Background: "#282828",
	},
}

// Styles holds all the UI styles
type Styles struct {
	Title     lipgloss.Style
	Normal    lipgloss.Style
	Help      lipgloss.Style
	Highlight lipgloss.Style
	Selected  lipgloss.Style
	Error     lipgloss.Style
	Success   lipgloss.Style
	theme     Theme
}

// NewStyles creates styles from a theme
func NewStyles(theme Theme) Styles {
	return Styles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(theme.Primary)).
			PaddingTop(1).
			PaddingBottom(1),

		Normal: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Text)),

		Help: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Help)).
			Italic(true),

		Highlight: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(theme.Highlight)),

		Selected: lipgloss.NewStyle().
			Background(lipgloss.Color(theme.Primary)).
			Foreground(lipgloss.Color(theme.Background)),

		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Error)),

		Success: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Success)),

		theme: theme,
	}
}

// DefaultStyles returns the default style set
func DefaultStyles() Styles {
	return NewStyles(Themes["default"])
}

// GetThemeNames returns available theme names
func GetThemeNames() []string {
	names := make([]string, 0, len(Themes))
	for name := range Themes {
		names = append(names, name)
	}
	return names
}
