package ui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Up             key.Binding
	Down           key.Binding
	CtrlD          key.Binding
	CtrlU          key.Binding
	ToggleFileTree key.Binding
	Search         key.Binding
	Quit           key.Binding
	Copy           key.Binding
	TogglePanel    key.Binding
}

var keys = &KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "prev file"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "next file"),
	),
	CtrlD: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", "diff down"),
	),
	CtrlU: key.NewBinding(
		key.WithKeys("ctrl+u"),
		key.WithHelp("ctrl+u", "diff up"),
	),
	ToggleFileTree: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "toggle file tree"),
	),
	Search: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "search files"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Copy: key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "copy file path"),
	),
	TogglePanel: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("TAB", "switch panel"),
	),
}

func getKeys() []key.Binding {
	return []key.Binding{
		keys.TogglePanel,
		keys.Up,
		keys.Down,
		keys.CtrlD,
		keys.CtrlU,
		keys.ToggleFileTree,
		keys.Search,
		keys.Copy,
		keys.Quit,
	}
}
