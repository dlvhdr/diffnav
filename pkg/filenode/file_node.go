package filenode

import (
	"path/filepath"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
)

// Icon style constants.
const (
	IconsNerdFonts    = "nerd-fonts"
	IconsNerdFontsAlt = "nerd-fonts-alt"
	IconsUnicode      = "unicode"
	IconsASCII        = "ascii"
)

type FileNode struct {
	File           *gitdiff.File
	Depth          int
	YOffset        int
	IconStyle      string
	Selected       bool
	ColorFileNames bool
	PanelWidth     int
}

func (f FileNode) Path() string {
	return GetFileName(f.File)
}

func (f FileNode) Value() string {
	icon := f.getIcon()
	name := filepath.Base(f.Path())
	// Icon is always colored by status
	coloredIcon := lipgloss.NewStyle().Foreground(f.StatusColor()).Render(icon)

	if f.Selected {
		// Apply background with fixed width to extend to panel edge
		bgStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(f.StatusColor()).
			Background(lipgloss.Color("#3a3a3a"))
		if f.PanelWidth > 0 {
			iconWidth := lipgloss.Width(coloredIcon) + 1
			// Subtract tree indentation
			availableWidth := f.PanelWidth - iconWidth - (f.Depth * 2)
			if availableWidth > 0 {
				bgStyle = bgStyle.Width(availableWidth)
			}
		}
		return coloredIcon + " " + bgStyle.Render(name)
	}

	if f.ColorFileNames {
		styledName := lipgloss.NewStyle().Foreground(f.StatusColor()).Render(name)
		return coloredIcon + " " + styledName
	}

	return coloredIcon + " " + name
}

func (f FileNode) getIcon() string {
	switch f.IconStyle {
	case IconsNerdFonts:
		if f.File.IsNew {
			return ""
		} else if f.File.IsDelete {
			return ""
		}
		return ""
	case IconsNerdFontsAlt:
		return ""
	case IconsUnicode:
		if f.File.IsNew {
			return "+"
		} else if f.File.IsDelete {
			return "⛌"
		}
		return "●"
	default: // ascii (fallback for unknown values)
		if f.File.IsNew {
			return "+"
		} else if f.File.IsDelete {
			return "x"
		}
		return "*"
	}
}

// StatusColor returns the color for this file based on its git status.
func (f FileNode) StatusColor() lipgloss.Color {
	if f.File.IsNew {
		return lipgloss.Color("2") // green
	} else if f.File.IsDelete {
		return lipgloss.Color("1") // red
	}
	return lipgloss.Color("3") // yellow/orange
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
