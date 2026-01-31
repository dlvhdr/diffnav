# diffnav

A git diff pager based on [delta](https://github.com/dandavison/delta) but with a file tree, à la GitHub.

<p align="center">
  <img width="900" src="https://github.com/user-attachments/assets/104e156e-7e9d-4ea5-bea1-399ca71e12a5" />
</p>


<details>
  <summary>Demo</summary>
  <img src="https://github.com/dlvhdr/diffnav/blob/74c3f341797ab121ce8edf785ef63e00075ce040/out.gif" />
</details>

## Installation

Homebrew:

```sh
brew install dlvhdr/formulae/diffnav
```

Go:

```sh
go install github.com/dlvhdr/diffnav@latest
```

> [!NOTE]
> To get the icons to render properly you should download and install a Nerd font from https://www.nerdfonts.com/. Then, select that font as your font for the terminal.
>
> _You can install these with brew as well: `brew install --cask font-<FONT NAME>-nerd-font`_

## Usage

### Pipe into diffnav

- `git diff | diffnav`
- `gh pr diff https://github.com/dlvhdr/gh-dash/pull/447 | diffnav`

### Set up as global git diff pager

```bash
git config --global pager.diff diffnav
```

## Flags

| Flag | Description |
|------|-------------|
| `--side-by-side, -s` | Force side-by-side diff view |
| `--unified, -u` | Force unified diff view |

Example:

```sh
git diff | diffnav --unified
git diff | diffnav -u
```

## Configuration

The config file is searched in this order:

1. `$DIFFNAV_CONFIG_DIR/config.yml` (if env var is set)
2. `$XDG_CONFIG_HOME/diffnav/config.yml` (if set, macOS only)
3. `~/.config/diffnav/config.yml` (macOS and Linux)
4. OS-specific config directory (e.g., `~/Library/Application Support/diffnav/config.yml` on macOS)

Example config file:

```yaml
ui:
  # Hide the header to get more screen space for diffs
  hideHeader: true

  # Hide the footer (keybindings help)
  hideFooter: true

  # Start with the file tree hidden (toggle with 'e')
  showFileTree: false

  # Customize the file tree width (default: 26)
  fileTreeWidth: 30

  # Customize the search panel width (default: 50)
  searchTreeWidth: 60

  # Icon style: "status" (default), "simple", "filetype", "full", "unicode", or "ascii"
  icons: nerd-fonts-status

  # Color filenames by git status (default: true)
  colorFileNames: false

  # Show the amount of lines added / removed next to the file
  showDiffStats: false

  # Use side-by-side diff view (default: true, set false for unified)
  sideBySide: true
```

| Option               | Type   | Default             | Description                                                                |
| :------------------- | :----- | :------------------ | :------------------------------------------------------------------------- |
| `ui.hideHeader`      | bool   | `false`             | Hide the "DIFFNAV" header                                                  |
| `ui.hideFooter`      | bool   | `false`             | Hide the footer with keybindings help                                      |
| `ui.showFileTree`    | bool   | `true`              | Show file tree on startup                                                  |
| `ui.fileTreeWidth`   | int    | `26`                | Width of the file tree sidebar                                             |
| `ui.searchTreeWidth` | int    | `50`                | Width of the search panel                                                  |
| `ui.icons`           | string | `nerd-fonts-status` | Icon style (see below for details)                                         |
| `ui.colorFileNames`  | bool   | `true`              | Color filenames by git status                                              |
| `ui.showDiffStats`   | bool   | `true`              | Show the amount of lines added / removed next to the file                  |
| `ui.sideBySide`      | bool   | `true`              | Use side-by-side diff view (false for unified) |

### Icon Styles

| Style                 | Description                                                      |
| :-------------------- | :--------------------------------------------------------------- |
| `nerd-fonts-status`   | Boxed git status icons colored by change type                    |
| `nerd-fonts-simple`   | Generic file icon colored by change type                         |
| `nerd-fonts-filetype` | File-type specific icons (language icons) colored by change type |
| `nerd-fonts-full`     | Both status icon and file-type icon, all colored                 |
| `unicode`             | Unicode symbols (+/⛌/●)                                          |
| `ascii`               | Plain ASCII characters (+/x/\*)                                  |

### Delta

You can also configure the diff rendering through delta. Check out [their docs](https://dandavison.github.io/delta/configuration.html).

If you want the exact delta configuration I'm using - [it can be found here](https://github.com/dlvhdr/diffnav/blob/main/cfg/delta.conf).

## Keys

| Key               | Description          |
| :---------------- | :------------------- |
| <kbd>j</kbd>      | Next file            |
| <kbd>k</kbd>      | Previous file        |
| <kbd>Ctrl-d</kbd> | Scroll the diff down |
| <kbd>Ctrl-u</kbd> | Scroll the diff up   |
| <kbd>e</kbd>      | Toggle the file tree |
| <kbd>t</kbd>      | Search/go-to file    |
| <kbd>y</kbd>      | Copy file path       |
| <kbd>i</kbd>      | Cycle icon style     |
| <kbd>o</kbd>      | Open file in $EDITOR |
| <kbd>s</kbd>      | Toggle side-by-side/unified view |
| <kbd>Tab</kbd>    | Switch focus between the panes |
| <kbd>q</kbd>      | Quit                 |

## Under the hood

`diffnav` uses:

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the TUI
- [`delta`](https://github.com/dandavison/delta) for viewing the diffed file

Screenshots use:

- [kitty](https://sw.kovidgoyal.net/kitty/) for the terminal
- [tokyonight](https://github.com/folke/tokyonight.nvim) for the color scheme
- [CommitMono](https://www.nerdfonts.com/font-downloads) for the font
