package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme represents the color theme for the application
type Theme struct {
	Primary    string
	Secondary  string
	Text       string
	Help       string
	Highlight  string
	Success    string
	Error      string
	Background string
	Subtle     string
}

// DefaultTheme is the single application theme
var DefaultTheme = Theme{
	Primary:    "#7D56F4",
	Secondary:  "#FAFAFA",
	Text:       "#FAFAFA",
	Help:       "#737373",
	Highlight:  "#04B575",
	Success:    "#04B575",
	Error:      "#FF0000",
	Background: "#000000",
	Subtle:     "#4A4A4A",
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

	HeaderBar lipgloss.Style
	FooterBar lipgloss.Style
	Border    lipgloss.Style
	Card      lipgloss.Style
	Detail    lipgloss.Style

	HelpKey  lipgloss.Style
	HelpDesc lipgloss.Style
	HelpSep  lipgloss.Style
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

		HeaderBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Primary)).
			Bold(true).
			PaddingLeft(1).
			PaddingRight(1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(subtle),

		FooterBar: lipgloss.NewStyle().
			PaddingLeft(1).
			PaddingRight(1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(subtle),

		Border: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(subtle).
			Padding(1, 2),

		Card: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.Primary)).
			Padding(1, 3),

		Detail: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Text)).
			PaddingLeft(1).
			PaddingRight(1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(subtle),

		HelpKey: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Primary)).
			Bold(true),

		HelpDesc: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Help)),

		HelpSep: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Subtle)),
	}
}

// DefaultStyles returns the default style set
func DefaultStyles() Styles {
	return NewStyles(DefaultTheme)
}
