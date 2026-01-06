# diffnav

A git diff pager based on [delta](https://github.com/dandavison/delta) but with a file tree, à la GitHub.

<p align="center">
  <img width="750" src="https://github.com/user-attachments/assets/3148be62-830a-484c-9256-2129ff10ca13" />
</p>

> [!CAUTION]
> This is early in development, bugs are to be expected.
>
> Feel free to open issues.

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
  hide_header: true

  # Hide the footer (keybindings help)
  hide_footer: true

  # Start with the file tree hidden (toggle with 'e')
  show_file_tree: false

  # Customize the file tree width (default: 26)
  file_tree_width: 30

  # Customize the search panel width (default: 50)
  search_tree_width: 60
```

| Option                 | Type | Default | Description                           |
| :--------------------- | :--- | :------ | :------------------------------------ |
| `ui.hide_header`       | bool | `false` | Hide the "DIFFNAV" header             |
| `ui.hide_footer`       | bool | `false` | Hide the footer with keybindings help |
| `ui.show_file_tree`    | bool | `true`  | Show file tree on startup             |
| `ui.file_tree_width`   | int  | `26`    | Width of the file tree sidebar        |
| `ui.search_tree_width` | int  | `50`    | Width of the search panel             |

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
| <kbd>q</kbd>      | Quit                 |

## Under the hood

`diffnav` uses:

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the TUI
- [`delta`](https://github.com/dandavison/delta) for viewing the diffed file

Screenshots use:

- [kitty](https://sw.kovidgoyal.net/kitty/) for the terminal
- [tokyonight](https://github.com/folke/tokyonight.nvim) for the color scheme
- [CommitMono](https://www.nerdfonts.com/font-downloads) for the font
