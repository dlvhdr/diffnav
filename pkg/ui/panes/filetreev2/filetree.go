package filetreev2

import (
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/tree"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	ltree "charm.land/lipgloss/v2/tree"
	"github.com/atotto/clipboard"
	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/charmbracelet/log"

	"github.com/dlvhdr/diffnav/pkg/constants"
	"github.com/dlvhdr/diffnav/pkg/filenode"
	"github.com/dlvhdr/diffnav/pkg/ui/common"
	"github.com/dlvhdr/diffnav/pkg/utils"
)

type Model struct {
	t              tree.Model
	files          []*gitdiff.File
	selectedFile   *string
	iconStyle      string
	colorFileNames bool
}

func New(iconStyle string, colorFileNames bool) Model {
	t := tree.New(nil, constants.OpenFileTreeWidth, 0)
	t.SetCursorCharacter("")
	t.SetShowHelp(false)
	t.SetOpenCharacter(getDirIcon(iconStyle))
	t.SetClosedCharacter(getDirIcon(iconStyle))
	t.Enumerator(enumerator).Indenter(indenter)
	t.SetScrollOff(3)

	m := Model{
		t:              t,
		iconStyle:      iconStyle,
		colorFileNames: colorFileNames,
	}
	m.updateStyles()

	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	return m, nil
}

func (m Model) View() string {
	return m.t.View()
}

func getDirIcon(iconStyle string) string {
	switch iconStyle {
	case filenode.IconsNerdStatus, filenode.IconsNerdSimple, filenode.IconsNerdFiletype, filenode.IconsNerdFull:
		return ""
	case filenode.IconsUnicode:
		return "▶"
	default: // ascii
		return ">"
	}
}

func (m *Model) updateStyles() {
	dimmed := lipgloss.Color("235")
	base := lipgloss.NewStyle()
	m.t.SetStyles(tree.Styles{
		TreeStyle:         base,
		RootNodeStyle:     base.Foreground(lipgloss.Color("4")),
		ParentNodeStyle:   base.Foreground(lipgloss.Color("4")),
		SelectedNodeStyle: base.Bold(true).Background(lipgloss.Color("8")),
		HelpStyle:         base.MarginTop(1),
		EnumeratorStyle:   base.Foreground(dimmed),
		IndenterStyle:     base.Foreground(dimmed),
	})

	m.t.SetOpenCharacter(getDirIcon(m.iconStyle))
	m.t.SetClosedCharacter(getDirIcon(m.iconStyle))
}

func (m Model) SetFiles(files []*gitdiff.File) Model {
	m.files = files
	m.rebuildTree()

	// need to re-apply the width as there's a bug where SetNodes resets it
	m.t.SetWidth(m.t.Width())
	m.updateStyles()

	return m
}

func (m *Model) SetCursor(cursor int) {
	if len(m.files) == 0 {
		return
	}
	name := filenode.GetFileName(m.files[cursor])
	m.selectedFile = &name
	m.rebuildTree()
	m.scrollSelectedFileIntoView(m.t.Root())
}

func (m *Model) rebuildTree() {
	t := buildFullFileTree(m.files, options{
		selectedFile:   m.selectedFile,
		iconStyle:      m.iconStyle,
		colorFileNames: m.colorFileNames,
	})
	t = collapseTree(t)
	t, _ = truncateTree(t, 0, 0, 0, m.iconStyle, m.selectedFile, m.colorFileNames, m.t.Width())
	m.t.SetNodes(t)
	m.t.SetWidth(m.t.Width())
	m.updateStyles()
}

func (m *Model) scrollSelectedFileIntoView(t *tree.Node) {
	children := t
	found := false
	for i := range len(children.ChildNodes()) {
		child := children.ChildNodes()[i]
		switch value := child.GivenValue().(type) {
		case string:
			m.scrollSelectedFileIntoView(child)
		case *filenode.FileNode:
			if value.Path() == *m.selectedFile {
				m.t.SetYOffset(child.YOffset())
				found = true
				break
			}
		}
		if found {
			break
		}
	}
}

type options struct {
	selectedFile   *string
	iconStyle      string
	colorFileNames bool
}

func buildFullFileTree(files []*gitdiff.File, opts options) *tree.Node {
	t := tree.Root(constants.RootName)
	for _, file := range files {
		// start from the root
		subTree := t

		name := filenode.GetFileName(file)
		dir := filepath.Dir(name)
		parts := strings.Split(dir, string(os.PathSeparator))
		path := ""

		// walk the tree to find existing path
		for _, part := range parts {
			found := false
			children := subTree.ChildNodes()
			for _, child := range children {
				if _, ok := child.GivenValue().(string); ok && child.GivenValue() == part {
					subTree = child
					path = path + part + string(os.PathSeparator)
					found = true
					// found a part of the path, continue to the subtree
					break
				}
			}
			if !found {
				break
			}
		}

		// path does not exist from this point, need to create it
		leftover := strings.TrimPrefix(name, path)
		parts = strings.Split(leftover, string(os.PathSeparator))
		for i, part := range parts {
			var c *tree.Node
			if i == len(parts)-1 {
				node := &filenode.FileNode{
					File:           file,
					IconStyle:      opts.iconStyle,
					ColorFileNames: opts.colorFileNames,
				}
				node.Selected = opts.selectedFile != nil && node.Path() == *opts.selectedFile
				subTree.Child(node)
			} else {
				c = tree.Root(part)
				subTree.Child(c)
				subTree = c
			}
		}
	}

	return t
}

// Given a tree with nodes that have only one child, collapse the tree by
// merging these nodes with their parents, as long as the parent has only one child as well.
// For example, the tree:
// .
// ├── a
// │   └── b
// │       └── c
//
// will be collapsed to:
// .
// ├── a/b
// │   └── c
//
// This tree wouldn't be collapsed:
// ├── a
// │   ├── b
// │   │   └── c
// │   └── d
func collapseTree(t *tree.Node) *tree.Node {
	children := t.ChildNodes()
	newT := tree.Root(t.GivenValue())
	if len(children) == 0 {
		return newT
	}

	// recursively collapse children
	for _, child := range children {
		if child.Size() > 1 {
			collapsedChild := collapseTree(child)
			newT.Child(collapsedChild)
		} else {
			newT.Child(child.GivenValue())
		}
	}

	// all children are collapsed, now check if the parent can be collapsed
	newChildren := newT.ChildNodes()
	if len(newChildren) == 1 {
		child := newChildren[0]
		// if the child is a string - it's a directory (files are filenode.FileNode).
		// So this tree has one chlid that's a directory -> collapse it
		if _, ok := child.GivenValue().(string); ok {

			// if the only child is a tree and its parent is the root we don't want to collapse.
			// The root should always be visible
			if newT.GivenValue() == constants.RootName {
				return newT
			}

			val := newT.GivenValue().(string) + string(os.PathSeparator) + child.GivenValue().(string)
			children := make([]any, 0)
			for _, c := range child.ChildNodes() {
				children = append(children, c)
			}

			collapsed := tree.Root(val).Child(children...)
			return collapsed
		}
	}

	return newT
}

func truncateTree(t *tree.Node, depth int, numNodes int, numChildren int, iconStyle string,
	selectedFile *string, colorFileNames bool, width int,
) (*tree.Node, int) {
	if _, ok := t.GivenValue().(string); !ok {
		return t, 0
	}

	newT := tree.Root(utils.TruncateString(t.GivenValue().(string), width-depth-2))
	numNodes++

	for _, child := range t.ChildNodes() {
		numChildren++
		switch value := child.GivenValue().(type) {
		case string:
			subTree, subNum := truncateTree(child, depth+1, numNodes, 0, iconStyle, selectedFile, colorFileNames, width)
			numChildren += subNum
			numNodes += subNum + 1
			child.SetValue(value)
			newT.Child(subTree)
		case *filenode.FileNode:
			numNodes++
			value.Depth = depth + 1
			value.PanelWidth = width
			newT.Child(value)
		}
	}
	return newT, numChildren
}

var enumerator = func(children ltree.Children, index int) string {
	return "│"
}

var indenter = func(children ltree.Children, index int) string {
	return "│"
}

// SetSize implements the Component interface.
func (m *Model) SetSize(width, height int) {
	log.Debug("setting tree size", "width", width, "height", height)
	m.t.SetSize(width, height)
	m.rebuildTree()
}

func (m *Model) Width() int {
	return m.t.Width()
}

// ViewportYOffset returns the tree's current Y scroll offset.
func (m Model) ViewportYOffset() int {
	return m.t.ViewportYOffset()
}

// GetYOffset returns the tree's current Y scroll offset.
func (m Model) GetFileAtY(yoffset int) string {
	node := m.t.Node(yoffset)
	if node == nil {
		return ""
	}

	if f, ok := node.GivenValue().(*filenode.FileNode); ok {
		return f.Path()
	}

	return ""
}

// SetCursorNoScroll updates the selected file without scrolling the viewport.
// Use this when the user clicks on a file they can already see.
func (m *Model) SetCursorNoScroll(cursor int) {
	if len(m.files) == 0 {
		return
	}
	name := filenode.GetFileName(m.files[cursor])
	m.selectedFile = &name
	scroll := m.t.ViewportYOffset()
	m.rebuildTree()

	for _, child := range m.t.AllNodes() {
		if f, ok := child.GivenValue().(*filenode.FileNode); ok && f.Path() == name {
			m.t.SetYOffset(child.YOffset())
			break
		}
	}
	m.t.SetViewportYOffset(scroll)
}

func (m *Model) CopyFilePath(cursor int) tea.Cmd {
	if len(m.files) == 0 {
		return nil
	}
	name := filenode.GetFileName(m.files[cursor])
	err := clipboard.WriteAll(name)
	if err != nil {
		return func() tea.Msg {
			return common.ErrMsg{Err: err}
		}
	}
	return nil
}

// ScrollUp scrolls the viewport up by the given number of lines.
func (m *Model) ScrollUp(lines int) {
	newOffset := m.t.ViewportYOffset() - lines
	m.t.SetViewportYOffset(newOffset)
}

// ScrollDown scrolls the viewport down by the given number of lines.
func (m *Model) ScrollDown(lines int) {
	newOffset := m.t.ViewportYOffset() + lines
	m.t.SetViewportYOffset(newOffset)
}

// SetIconStyle changes the icon style and regenerates the tree.
func (m *Model) SetIconStyle(iconStyle string) {
	m.iconStyle = iconStyle
	if len(m.files) > 0 {
		m.rebuildTree()
	}
	m.updateStyles()
}
