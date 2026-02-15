package ui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the keybindings for the application
type KeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Left   key.Binding
	Right  key.Binding
	Enter  key.Binding
	Back   key.Binding
	Quit   key.Binding
	Help   key.Binding
	Select key.Binding
	Open   key.Binding
	Update key.Binding
}

// DefaultKeyMap returns the default keybindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "right"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "backspace"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Select: key.NewBinding(
			key.WithKeys("x", " ", "space"),
			key.WithHelp("x/space", "toggle select"),
		),
		Open: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "open url"),
		),
		Update: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "update readwise"),
		),
	}
}

// Keys returns the keys as a slice for matching
func (k KeyMap) Keys() []key.Binding {
	return []key.Binding{
		k.Up, k.Down, k.Left, k.Right,
		k.Enter, k.Back, k.Quit, k.Help, k.Select, k.Open, k.Update,
	}
}
