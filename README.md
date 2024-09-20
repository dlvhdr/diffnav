# diffnav

A git diff pager based on [delta](https://github.com/dandavison/delta) but with a file tree, à la GitHub.

<p align="center">
  <img width="750" src="https://github.com/user-attachments/assets/1b04ff0d-3054-42b4-8a27-50f338d2aa26" />
</p>

<details>
  <summary>Demo</summary>
>
  <img src="https://github.com/dlvhdr/diffnav/blob/74c3f341797ab121ce8edf785ef63e00075ce040/out.gif" />
</details>

## Installation

```bash
brew install git-delta # or any other package manager
go install github.com/dlvhdr/diffnav
```

- [See here](https://dandavison.github.io/delta/installation.html) the full delta installations instructions.
- _TBD: support for package managers_

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
