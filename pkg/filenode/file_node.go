package filenode

import (
	"path/filepath"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"

	"github.com/dlvhdr/diffnav/pkg/constants"
	"github.com/dlvhdr/diffnav/pkg/utils"
)

type FileNode struct {
	File    *gitdiff.File
	Depth   int
	YOffset int
}

func (f FileNode) Path() string {
	return GetFileName(f.File)
}

func (f FileNode) Value() string {
	icon := " "
	status := " "
	if f.File.IsNew {
		status += lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render("")
	} else if f.File.IsDelete {
		status += lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("")
	} else {
		status += lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("")
	}

	depthWidth := f.Depth * 2
	iconsWidth := lipgloss.Width(icon) + lipgloss.Width(status)
	nameMaxWidth := constants.OpenFileTreeWidth - depthWidth - iconsWidth
	base := filepath.Base(f.Path())
	name := utils.TruncateString(base, nameMaxWidth)

	spacerWidth := constants.OpenFileTreeWidth - lipgloss.Width(name) - iconsWidth - depthWidth
	if len(name) < len(base) {
		spacerWidth = spacerWidth - 1
	}
	spacer := ""
	if spacerWidth > 0 {
		spacer = strings.Repeat(" ", spacerWidth)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, icon, name, spacer, status)
}

func (f FileNode) String() string {
	return f.Value()
}

func (f FileNode) Children() tree.Children {
	return tree.NodeChildren(nil)
}

func (f FileNode) Hidden() bool {
	return false
}

func (f FileNode) SetHidden(bool) {}

func (f FileNode) SetValue(any) {}

func GetFileName(file *gitdiff.File) string {
	if file.NewName != "" {
		return file.NewName
	}
	return file.OldName
}
