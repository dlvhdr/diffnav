package ui

import "charm.land/bubbles/v2/key"

type KeyMap struct {
	ExpandNode      key.Binding
	CollapseNode    key.Binding
	ToggleNode      key.Binding
	Up              key.Binding
	Down            key.Binding
	CtrlD           key.Binding
	CtrlU           key.Binding
	ToggleFileTree  key.Binding
	Search          key.Binding
	Quit            key.Binding
	Copy            key.Binding
	SwitchPanel     key.Binding
	OpenInEditor    key.Binding
	ToggleDiffView  key.Binding
	ToggleIconStyle key.Binding
}

var keys = &KeyMap{
	ExpandNode: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "expand"),
	),
	CollapseNode: key.NewBinding(
		key.WithKeys("h"),
		key.WithHelp("h", "collapse"),
	),
	ToggleNode: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "toggle"),
	),
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
	SwitchPanel: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("TAB", "switch panel"),
	),
	OpenInEditor: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open"),
	),
	ToggleDiffView: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "toggle side-by-side"),
	),
	ToggleIconStyle: key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "toggle ToggleIconStyle"),
	),
}

func getKeys() []key.Binding {
	return []key.Binding{
		keys.SwitchPanel,
		keys.Up,
		keys.Down,
		keys.CtrlD,
		keys.CtrlU,
		keys.ToggleFileTree,
		keys.Search,
		keys.Copy,
		keys.OpenInEditor,
		keys.ToggleDiffView,
		keys.ToggleIconStyle,
		keys.Quit,
	}
}
