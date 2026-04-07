package ui

import "charm.land/bubbles/v2/key"

type keyMap struct {
	Up          key.Binding
	Down        key.Binding
	Enter       key.Binding
	Right       key.Binding
	Left        key.Binding
	Search      key.Binding
	Tab         key.Binding
	Reload      key.Binding
	Sort        key.Binding
	Select      key.Binding
	SelectAll   key.Binding
	SelectAllUp key.Binding
	Info        key.Binding
	Escape      key.Binding
	Quit        key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("↑", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("↓", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "update"),
	),
	Right: key.NewBinding(
		key.WithKeys("right"),
		key.WithHelp("→", "versions"),
	),
	Left: key.NewBinding(
		key.WithKeys("left"),
		key.WithHelp("←", "back"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch search mode"),
	),
	Reload: key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("ctrl+r", "reload index"),
	),
	Sort: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "sort"),
	),
	Select: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "select"),
	),
	SelectAll: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "select outdated"),
	),
	SelectAllUp: key.NewBinding(
		key.WithKeys("A"),
		key.WithHelp("A", "select all"),
	),
	Info: key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "package info"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
