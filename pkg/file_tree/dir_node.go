package filetree

import "github.com/charmbracelet/lipgloss/tree"

type DirNode struct {
	*tree.Tree
}

func (d *DirNode) Value() string {
	return "X " + d.Tree.Value()
}
