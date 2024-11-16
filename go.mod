module github.com/dlvhdr/diffnav

go 1.24.2

toolchain go1.24.11

require (
	charm.land/bubbles/v2 v2.0.0-rc.1
	charm.land/bubbletea/v2 v2.0.0-rc.1.0.20251116163936-f60735dff10c
	charm.land/lipgloss/v2 v2.0.0-beta.3.0.20251205162909-7869489d8971
	github.com/atotto/clipboard v0.1.4
	github.com/bluekeyes/go-gitdiff v0.8.0
	github.com/charmbracelet/log v0.4.0
	github.com/charmbracelet/x/ansi v0.11.2
	github.com/lrstanley/bubblezone/v2 v2.0.0-alpha.3
	github.com/muesli/reflow v0.3.0
	github.com/muesli/termenv v0.16.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/charmbracelet/colorprofile v0.3.3 // indirect
	github.com/charmbracelet/lipgloss v0.10.0 // indirect
	github.com/charmbracelet/ultraviolet v0.0.0-20251205161215-1948445e3318 // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/charmbracelet/x/termios v0.1.1 // indirect
	github.com/charmbracelet/x/windows v0.2.2 // indirect
	github.com/clipperhouse/displaywidth v0.6.0 // indirect
	github.com/clipperhouse/stringish v0.1.1 // indirect
	github.com/clipperhouse/uax29/v2 v2.3.0 // indirect
	github.com/go-logfmt/logfmt v0.6.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.3.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.19 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	golang.org/x/exp v0.0.0-20231006140011-7918f672742d // indirect
	golang.org/x/sync v0.18.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
)

replace charm.land/bubbles/v2 v2.0.0-rc.1 => github.com/dlvhdr/bubbles/v2 v2.0.0-rc-dlvhdr-2

replace charm.land/lipgloss/v2 v2.0.0-beta.3.0.20251114164805-d267651963ad => charm.land/lipgloss/v2 v2.0.0-beta.3.0.20251205162909-7869489d8971
