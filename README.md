<br />
<p align="center">
  <img width="504" height="96" alt="output-onlinepngtools" src="https://github.com/user-attachments/assets/b932225f-7f49-4274-886d-61e640f4ef8b" />  
</p>

<p align="center">
  A git diff pager based on <a href="https://github.com/dandavison/delta">delta</a> but with a file tree, à la GitHub.
  <br />
  <br />
  <a href="https://github.com/dlvhdr/gh-dash/releases"><img src="https://img.shields.io/github/release/dlvhdr/diffnav.svg" alt="Latest Release"></a>
  <a href="https://discord.gg/SXNXp9NctV"><img src="https://img.shields.io/discord/1413193703476035755?label=discord" alt="Discord"/></a>
  <a href="https://github.com/sponsors/dlvhdr"><img src=https://img.shields.io/github/sponsors/dlvhdr?logo=githubsponsors&color=EA4AAA /></a>
  <a href="https://www.jetify.com/devbox/docs/contributor-quickstart/" alt="Built with Devbox"><img src="https://www.jetify.com/img/devbox/shield_galaxy.svg" /></a>
</p>

<p align="center">
  <img width="900" src="https://github.com/user-attachments/assets/104e156e-7e9d-4ea5-bea1-399ca71e12a5" />
</p>

## Donating ❤️

If you enjoy `diffnav` and want to help, consider supporting the project with a
donation at the [sponsors page](https://github.com/sponsors/dlvhdr).

## Installation

Homebrew:

```sh
brew install dlvhdr/formulae/diffnav
```

Go:

```sh
git clone https://github.com/dlvhdr/diffnav.git
cd diffnav
go install .
```

> [!NOTE]
> To get the icons to render properly you should download and install a Nerd font from https://www.nerdfonts.com/. Then, select that font as your font for the terminal.
>
> _You can install these with brew as well: `brew install --cask font-<FONT NAME>-nerd-font`_

## Usage

### Pipe into `diffnav`

- `git diff | diffnav`
- `gh pr diff https://github.com/dlvhdr/gh-dash/pull/447 | diffnav`

### Set up as Global Git Diff Pager

```bash
git config --global pager.diff diffnav
```

## Flags

| Flag                 | Description                  |
| -------------------- | ---------------------------- |
| `--side-by-side, -s` | Force side-by-side diff view |
| `--unified, -u`      | Force unified diff view      |

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

| Option               | Type   | Default             | Description                                               |
| :------------------- | :----- | :------------------ | :-------------------------------------------------------- |
| `ui.hideHeader`      | bool   | `false`             | Hide the "DIFFNAV" header                                 |
| `ui.hideFooter`      | bool   | `false`             | Hide the footer with keybindings help                     |
| `ui.showFileTree`    | bool   | `true`              | Show file tree on startup                                 |
| `ui.fileTreeWidth`   | int    | `26`                | Width of the file tree sidebar                            |
| `ui.searchTreeWidth` | int    | `50`                | Width of the search panel                                 |
| `ui.icons`           | string | `nerd-fonts-status` | Icon style (see below for details)                        |
| `ui.colorFileNames`  | bool   | `true`              | Color filenames by git status                             |
| `ui.showDiffStats`   | bool   | `true`              | Show the amount of lines added / removed next to the file |
| `ui.sideBySide`      | bool   | `true`              | Use side-by-side diff view (false for unified)            |

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

| Key               | Description                      |
| :---------------- | :------------------------------- |
| <kbd>j</kbd>      | Next file                        |
| <kbd>k</kbd>      | Previous file                    |
| <kbd>Ctrl-d</kbd> | Scroll the diff down             |
| <kbd>Ctrl-u</kbd> | Scroll the diff up               |
| <kbd>e</kbd>      | Toggle the file tree             |
| <kbd>t</kbd>      | Search/go-to file                |
| <kbd>y</kbd>      | Copy file path                   |
| <kbd>i</kbd>      | Cycle icon style                 |
| <kbd>o</kbd>      | Open file in $EDITOR             |
| <kbd>s</kbd>      | Toggle side-by-side/unified view |
| <kbd>Tab</kbd>    | Switch focus between the panes   |
| <kbd>q</kbd>      | Quit                             |

## Discord

Have questions? Join our [Discord community](https://discord.gg/SXNXp9NctV)!

## Contributing

See the contribution guide at [https://www.gh-dash.dev/contributing](https://www.gh-dash.dev/contributing/).

## Under the Hood

`diffnav` uses:

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the TUI
- [`delta`](https://github.com/dandavison/delta) for viewing the diffed file

Screenshots use:

- [kitty](https://sw.kovidgoyal.net/kitty/) for the terminal
- [tokyonight](https://github.com/folke/tokyonight.nvim) for the color scheme
- [CommitMono](https://www.nerdfonts.com/font-downloads) for the font
