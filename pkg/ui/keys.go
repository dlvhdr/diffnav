package ui

import "charm.land/bubbles/v2/key"

type KeyMap struct {
	ExpandNode            key.Binding
	CollapseNode          key.Binding
	ToggleNode            key.Binding
	Up                    key.Binding
	Down                  key.Binding
	Bottom                key.Binding
	Top                   key.Binding
	NextFile              key.Binding
	PrevFile              key.Binding
	CtrlD                 key.Binding
	CtrlU                 key.Binding
	CtrlE                 key.Binding
	CtrlY                 key.Binding
	ScrollLeft            key.Binding
	ScrollRight           key.Binding
	ToggleFileTree        key.Binding
	SearchFiles           key.Binding
	SearchDiffExact       key.Binding
	SearchDiffInsensitive key.Binding
	SearchDiffRegex       key.Binding
	SearchDiffFuzzy       key.Binding
	Quit                  key.Binding
	Copy                  key.Binding
	SwitchPanel           key.Binding
	OpenInEditor          key.Binding
	ToggleDiffView        key.Binding
	ToggleIconStyle       key.Binding
	ToggleHelp            key.Binding
	ToggleMessage         key.Binding
	ToggleSelection       key.Binding
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
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "prev file"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "next file"),
	),
	Bottom: key.NewBinding(
		key.WithKeys("G"),
		key.WithHelp("G", "bottom"),
	),
	Top: key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("g", "top"),
	),
	NextFile: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "next file"),
	),
	PrevFile: key.NewBinding(
		key.WithKeys("p", "N"),
		key.WithHelp("p/N", "prev file"),
	),
	CtrlD: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", "diff half page down"),
	),
	CtrlU: key.NewBinding(
		key.WithKeys("ctrl+u"),
		key.WithHelp("ctrl+u", "diff half page up"),
	),
	CtrlE: key.NewBinding(
		key.WithKeys("ctrl+e"),
		key.WithHelp("ctrl+e", "diff line down"),
	),
	CtrlY: key.NewBinding(
		key.WithKeys("ctrl+y"),
		key.WithHelp("ctrl+y", "diff line up"),
	),
	ScrollLeft: key.NewBinding(
		key.WithKeys("left"),
		key.WithHelp("←", "scroll left"),
	),
	ScrollRight: key.NewBinding(
		key.WithKeys("right"),
		key.WithHelp("→", "scroll right"),
	),
	ToggleFileTree: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "toggle file tree"),
	),
	SearchFiles: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "search files"),
	),
	SearchDiffExact: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search diff (exact)"),
	),
	SearchDiffInsensitive: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "search diff (case insensitive)"),
	),
	SearchDiffRegex: key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("ctrl+r", "search diff (regex)"),
	),
	SearchDiffFuzzy: key.NewBinding(
		key.WithKeys("ctrl+f"),
		key.WithHelp("ctrl+f", "search diff (fuzzy)"),
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
		key.WithKeys("tab", "shift+tab"),
		key.WithHelp("tab", "switch panel"),
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
		key.WithHelp("i", "toggle icon style"),
	),
	ToggleHelp: key.NewBinding(
		key.WithKeys("?", "f1"),
		key.WithHelp("F1/?", "toggle help"),
	),
	ToggleMessage: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "commit info"),
	),
	ToggleSelection: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "toggle selection"),
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

func KeyGroups() [][]key.Binding {
	return [][]key.Binding{{
		keys.SwitchPanel,
		keys.Up,
		keys.Down,
		keys.Top,
		keys.Bottom,
		keys.NextFile,
		keys.PrevFile,
		keys.CtrlD,
		keys.CtrlU,
		keys.ScrollLeft,
		keys.ScrollRight,
		keys.IncreaseFileTreeWidth,
	}, {
		keys.ToggleFileTree,
		keys.SearchFiles,
		keys.SearchDiffExact,
		keys.SearchDiffInsensitive,
		keys.SearchDiffRegex,
		keys.SearchDiffFuzzy,
		keys.Copy,
		keys.OpenInEditor,
		keys.ToggleDiffView,
		keys.ToggleIconStyle,
		keys.ToggleSelection,
		keys.DecreaseFileTreeWidth,
	}, {
		keys.ToggleMessage,
		keys.ToggleHelp,
		keys.Quit,
	}}
}
