# diffnav

A git diff pager based on [delta](https://github.com/dandavison/delta) but with a file tree, à la GitHub.

<p align="center">
  <img width="750" src="https://github.com/user-attachments/assets/359cd2a3-a22f-4572-8a09-aa57befadd5d" />
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
go install github.com/dlvhdr/diffnav
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

- Currently you can configure `diffnav` only through delta so [check out their docs](https://dandavison.github.io/delta/configuration.html).
- If you want the exact configuration I'm using - [it can be found here](https://github.com/dlvhdr/diffnav/blob/main/cfg/delta.conf).

## Keys

- <kbd>j</kbd>/<kbd>k</kbd> - navigate the file tree
- <kbd>Ctrl-d</kbd>/<kbd>Ctrl-u</kbd> - navigate the diff
- <kbd>e</kbd> - toggle the file tree
- <kbd>q</kbd>/<kbd>Ctrl+c</kbd> - quit
