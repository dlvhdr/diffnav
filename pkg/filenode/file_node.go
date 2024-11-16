package filenode

import (
	"image/color"
	"path/filepath"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/tree"
	"github.com/bluekeyes/go-gitdiff/gitdiff"

	"github.com/dlvhdr/diffnav/pkg/icons"
	"github.com/dlvhdr/diffnav/pkg/utils"
)

// Icon style constants.
const (
	IconsNerdStatus   = "nerd-fonts-status"
	IconsNerdSimple   = "nerd-fonts-simple"
	IconsNerdFiletype = "nerd-fonts-filetype"
	IconsNerdFull     = "nerd-fonts-full"
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

func (f *FileNode) Path() string {
	return GetFileName(f.File)
}

func (f *FileNode) Value() string {
	name := filepath.Base(f.Path())

	// full has a special layout: [status icon] [filename] [file-type icon]
	if f.IconStyle == IconsNerdFull {
		return utils.RemoveReset(f.renderFullLayout(name))
	}

	// All other styles: [icon] [filename] with optional coloring
	return utils.RemoveReset(f.renderStandardLayout(name))
}

// renderStandardLayout renders: [icon colored] [filename]
// Used by status, simple, filetype, unicode, ascii.
func (f *FileNode) renderStandardLayout(name string) string {
	icon := f.getIcon() + " "
	iconWidth := lipgloss.Width(icon) + 1
	nameMaxWidth := f.PanelWidth - f.Depth - iconWidth
	truncatedName := utils.TruncateString(name, nameMaxWidth)
	coloredIcon := lipgloss.NewStyle().Foreground(f.StatusColor()).Render(icon)

	if f.Selected {
		bgStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(f.StatusColor())
		if f.PanelWidth > 0 {
			availableWidth := f.PanelWidth - iconWidth - f.Depth
			if availableWidth > 0 {
				bgStyle = bgStyle.Width(availableWidth)
			}
		}
		return coloredIcon + bgStyle.Render(truncatedName)
	}

	if f.ColorFileNames {
		styledName := lipgloss.NewStyle().Foreground(f.StatusColor()).Render(truncatedName)
		return coloredIcon + styledName
	}

	return coloredIcon + truncatedName
}

// renderFullLayout renders: [status icon colored] [file-type icon colored] [filename]
// All icons colored by git status.
func (f *FileNode) renderFullLayout(name string) string {
	statusIcon := f.getStatusIcon()
	fileIcon := icons.GetIcon(name, false)
	style := lipgloss.NewStyle().Foreground(f.StatusColor())

	iconsPrefix := style.Render(statusIcon) + " " + style.Render(fileIcon) + " "
	iconsWidth := lipgloss.Width(statusIcon) + 1 + lipgloss.Width(fileIcon) + 1

	nameMaxWidth := f.PanelWidth - f.Depth - iconsWidth
	truncatedName := utils.TruncateString(name, nameMaxWidth)

	if f.Selected {
		bgStyle := style.Bold(true)
		if f.PanelWidth > 0 {
			if w := f.PanelWidth - iconsWidth - f.Depth; w > 0 {
				bgStyle = bgStyle.Width(w)
			}
		}
		return iconsPrefix + bgStyle.Render(truncatedName)
	}

	if f.ColorFileNames {
		return iconsPrefix + style.Render(truncatedName)
	}
	return iconsPrefix + lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render(truncatedName)
}

// getIcon returns the left icon based on the icon style.
func (f *FileNode) getIcon() string {
	name := filepath.Base(f.Path())
	switch f.IconStyle {
	case IconsNerdStatus:
		if f.File.IsNew {
			return ""
		} else if f.File.IsDelete {
			return ""
		}
		return ""
	case IconsNerdSimple:
		return ""
	case IconsNerdFiletype:
		return icons.GetIcon(name, false) // File-type specific icon (colored by status)
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

// getStatusIcon returns the git status indicator icon (used by full layout).
// Uses the same boxed icons as status style.
func (f *FileNode) getStatusIcon() string {
	if f.File.IsNew {
		return "\uf457" //
	} else if f.File.IsDelete {
		return "\ueadf" //
	}
	return "\uf459" //
}

// StatusColor returns the color for this file based on its git status.
func (f *FileNode) StatusColor() color.Color {
	if f.File.IsNew {
		return lipgloss.Color("2") // green
	} else if f.File.IsDelete {
		return lipgloss.Color("1") // red
	}
	return lipgloss.Color("3") // yellow/orange
}

func (f *FileNode) String() string {
	return f.Value()
}

func (f *FileNode) Children() tree.Children {
	return tree.NodeChildren(nil)
}

func (f *FileNode) Hidden() bool {
	return false
}

func (f *FileNode) SetHidden(bool) {}

func (f *FileNode) SetValue(any) {}

func GetFileName(file *gitdiff.File) string {
	if file.NewName != "" {
		return file.NewName
	}
	return file.OldName
}
