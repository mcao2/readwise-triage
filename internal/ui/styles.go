package ui

import (
	"sort"

	"github.com/charmbracelet/lipgloss"
)

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
	Subtle     string // Dimmed text for borders, separators
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
		Subtle:     "#4A4A4A",
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
		Subtle:     "#45475A",
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
		Subtle:     "#44475A",
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
		Subtle:     "#3B4252",
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
		Subtle:     "#3C3836",
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

	// Layout styles
	HeaderBar lipgloss.Style
	FooterBar lipgloss.Style
	Border    lipgloss.Style
	Card      lipgloss.Style
	Detail    lipgloss.Style

	// Help key styles
	HelpKey  lipgloss.Style
	HelpDesc lipgloss.Style
	HelpSep  lipgloss.Style

	theme Theme
}

// NewStyles creates styles from a theme
func NewStyles(theme Theme) Styles {
	subtle := lipgloss.Color(theme.Subtle)

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

		// Header bar: primary color text, bottom border
		HeaderBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Primary)).
			Bold(true).
			PaddingLeft(1).
			PaddingRight(1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(subtle),

		// Footer bar: help text area with top border
		FooterBar: lipgloss.NewStyle().
			PaddingLeft(1).
			PaddingRight(1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(subtle),

		// Rounded border for panels
		Border: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(subtle).
			Padding(1, 2),

		// Card style for config screen
		Card: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.Primary)).
			Padding(1, 3),

		// Detail pane below table
		Detail: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Text)).
			PaddingLeft(1).
			PaddingRight(1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(subtle),

		// Help key styling
		HelpKey: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Primary)).
			Bold(true),

		HelpDesc: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Help)),

		HelpSep: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Subtle)),

		theme: theme,
	}
}

// DefaultStyles returns the default style set
func DefaultStyles() Styles {
	return NewStyles(Themes["default"])
}

func GetThemeNames() []string {
	names := make([]string, 0, len(Themes))
	for name := range Themes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
