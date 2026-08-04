package filetree

import "charm.land/bubbles/v2/key"

type KeyMap struct {
	ExpandNode            key.Binding
	CollapseNode          key.Binding
	ToggleNode            key.Binding
	IncreaseFileTreeWidth key.Binding
	DecreaseFileTreeWidth key.Binding
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
	IncreaseFileTreeWidth: key.NewBinding(
		key.WithKeys(">"),
		key.WithHelp(">", "increase file tree width"),
	),
	DecreaseFileTreeWidth: key.NewBinding(
		key.WithKeys("<"),
		key.WithHelp("<", "decrease file tree width"),
	),
}
