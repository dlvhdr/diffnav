package main

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Up             key.Binding
	Down           key.Binding
	CtrlD          key.Binding
	CtrlU          key.Binding
	ToggleFileTree key.Binding
	Search         key.Binding
	Quit           key.Binding
}

var keys = &KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "K"),
		key.WithHelp("↑/K", "prev file"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "J"),
		key.WithHelp("↓/J", "next file"),
	),
	CtrlD: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("j/ctrl+d", "diff down"),
	),
	CtrlU: key.NewBinding(
		key.WithKeys("ctrl+u"),
		key.WithHelp("k/ctrl+u", "diff up"),
	),
	ToggleFileTree: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "toggle file tree"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search files"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

func getKeys() []key.Binding {
	return []key.Binding{keys.Up, keys.Down, keys.CtrlD, keys.CtrlU, keys.ToggleFileTree, keys.Search, keys.Quit}
}
