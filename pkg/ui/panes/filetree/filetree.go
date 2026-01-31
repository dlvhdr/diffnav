package filetree

import (
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/tree"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	ltree "charm.land/lipgloss/v2/tree"
	"github.com/atotto/clipboard"
	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/charmbracelet/log"

	"github.com/dlvhdr/diffnav/pkg/constants"
	"github.com/dlvhdr/diffnav/pkg/dirnode"
	"github.com/dlvhdr/diffnav/pkg/filenode"
	"github.com/dlvhdr/diffnav/pkg/ui/common"
	"github.com/dlvhdr/diffnav/pkg/utils"
)

type Model struct {
	t              tree.Model
	files          []*gitdiff.File
	iconStyle      string
	colorFileNames bool
}

func New(iconStyle string, colorFileNames bool) Model {
	t := tree.New(nil, constants.OpenFileTreeWidth, 0)
	t.SetCursorCharacter("")
	t.SetShowHelp(false)
	t.Enumerator(enumerator).Indenter(indenter)
	t.SetScrollOff(3)

	m := Model{
		t:              t,
		iconStyle:      iconStyle,
		colorFileNames: colorFileNames,
	}

	open, closed := getDirIcons(m.iconStyle)
	t.SetOpenCharacter(open)
	t.SetClosedCharacter(closed)
	m.updateStyles()

	return m
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keys.ExpandNode):
			m.t.OpenCurrentNode()

		case key.Matches(msg, keys.CollapseNode):
			m.t.CloseCurrentNode()

		case key.Matches(msg, keys.ToggleNode):
			m.t.ToggleCurrentNode()
		}
	}
	return m, nil
}

func (m *Model) View() string {
	return m.t.View()
}

func getDirIcons(iconStyle string) (string, string) {
	switch iconStyle {
	case filenode.IconsNerdStatus, filenode.IconsNerdSimple, filenode.IconsNerdFiletype, filenode.IconsNerdFull:
		return "", "󰉋"
	case filenode.IconsUnicode:
		return "▼", "▶"
	default: // ascii
		return ">", "-"
	}
}

func (m *Model) updateStyles() {
	dimmed := common.Colors[common.Selected]
	base := lipgloss.NewStyle()
	m.t.SetStyles(tree.Styles{
		TreeStyle:       base,
		RootNodeStyle:   base.Foreground(lipgloss.BrightBlue),
		ParentNodeStyle: base.Foreground(lipgloss.BrightBlue),
		SelectedNodeStyleFunc: func(children tree.Nodes, i int) lipgloss.Style {
			base := base.Bold(true).Background(dimmed)
			child := children.At(i)
			switch child.GivenValue().(type) {
			case *filenode.FileNode:
				return base
			case string, *dirnode.DirNode:
				return base.Foreground(lipgloss.BrightBlue)
			}
			return base
		},
		HelpStyle:               base.MarginTop(1),
		EnumeratorStyle:         base.Foreground(dimmed),
		SelectedEnumeratorStyle: base.Bold(true).Foreground(lipgloss.BrightBlue),
		IndenterStyle:           base.Foreground(dimmed),
	})

	open, closed := getDirIcons(m.iconStyle)
	m.t.SetOpenCharacter(open)
	m.t.SetClosedCharacter(closed)
}

func (m Model) SetFiles(files []*gitdiff.File) Model {
	m.files = files
	m.rebuildTree()

	// need to re-apply the width as there's a bug where SetNodes resets it
	m.t.SetWidth(m.t.Width())
	m.updateStyles()

	return m
}

func (m *Model) Down() {
	m.t.Down()
}

func (m *Model) Up() {
	m.t.Up()
}

func (m *Model) SetCursorByPath(path string) {
	if len(m.files) == 0 {
		return
	}

	yoffset := 0
	for _, node := range m.t.AllNodes() {
		switch val := node.GivenValue().(type) {
		case *filenode.FileNode:
			name := filenode.GetFileName(val.File)
			if name == path {
				yoffset = node.YOffset()
				break
			}
		case *dirnode.DirNode:
			if val.FullPath == path {
				yoffset = node.YOffset()
				break
			}
		}
	}

	log.Debug("filetree - setting cursor", "name", path)
	m.rebuildTree()
	m.t.SetYOffset(yoffset)
}

func (m *Model) rebuildTree() {
	t := buildFullFileTree(m.files, options{
		iconStyle:      m.iconStyle,
		colorFileNames: m.colorFileNames,
	})
	t = collapseTree(t)
	t, _ = truncateTree(t, 0, 0, 0, m.iconStyle, m.colorFileNames, m.t.Width())
	m.t.SetNodes(t)
	m.t.SetWidth(m.t.Width())
	m.updateStyles()
}

type options struct {
	iconStyle      string
	colorFileNames bool
}

func buildFullFileTree(files []*gitdiff.File, opts options) *tree.Node {
	t := tree.Root(&dirnode.DirNode{FullPath: "/", Name: constants.RootName})
	for _, file := range files {
		// start from the root
		subTree := t

		name := filenode.GetFileName(file)
		dir := filepath.Dir(name)
		parts := strings.Split(dir, string(os.PathSeparator))
		existingPath := ""

		// walk the tree to find existing path
		for _, part := range parts {
			found := false
			children := subTree.ChildNodes()
			for _, child := range children {
				if dir, ok := child.GivenValue().(*dirnode.DirNode); ok && dir.Name == part {
					subTree = child
					existingPath = existingPath + part + string(os.PathSeparator)
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
		leftover := strings.TrimPrefix(name, existingPath)
		parts = strings.Split(leftover, string(os.PathSeparator))
		for i, part := range parts {
			var c *tree.Node
			if i == len(parts)-1 {
				node := &filenode.FileNode{
					File:           file,
					IconStyle:      opts.iconStyle,
					ColorFileNames: opts.colorFileNames,
				}
				subTree.Child(node)
			} else {
				dirNode := dirnode.DirNode{
					Name:     part,
					FullPath: filepath.Join(existingPath, filepath.Join(parts[:i]...), part),
				}
				c = tree.Root(&dirNode)
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
	rootDir, ok := t.GivenValue().(*dirnode.DirNode)
	if !ok {
		log.Fatalf("failed collapsing tree, root is not a directory")
	}
	newT := tree.Root(rootDir)
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
		// If the child is dir with one chlid that's also a dir -> collapse it
		if dir, ok := child.GivenValue().(*dirnode.DirNode); ok {

			// if the only child is a tree and its parent is the root we don't want to collapse.
			// The root should always be visible
			if rootDir.Name == constants.RootName {
				return newT
			}

			newDir := dirnode.DirNode{
				FullPath: filepath.Join(rootDir.FullPath, dir.Name),
				Name:     filepath.Join(rootDir.Name, dir.Name),
			}
			children := make([]any, 0)
			for _, c := range child.ChildNodes() {
				children = append(children, c)
			}

			collapsed := tree.Root(&newDir).Child(children...)
			return collapsed
		}
	}

	return newT
}

func truncateTree(t *tree.Node, depth int, numNodes int, numChildren int, iconStyle string,
	colorFileNames bool, width int,
) (*tree.Node, int) {
	dir, ok := t.GivenValue().(*dirnode.DirNode)
	if !ok {
		return t, 0
	}

	newT := tree.Root(&dirnode.DirNode{Name: utils.TruncateString(dir.Name, width-depth-2), FullPath: dir.FullPath})
	numNodes++

	for _, child := range t.ChildNodes() {
		numChildren++
		switch value := child.GivenValue().(type) {
		case *dirnode.DirNode:
			subTree, subNum := truncateTree(child, depth+1, numNodes, 0, iconStyle, colorFileNames, width)
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

// GetNodeAtY returns the tree's node at the Y scroll offset.
func (m Model) GetNodeAtY(yoffset int) *tree.Node {
	return m.t.Node(yoffset)
}

func (m *Model) GetCurrNode() *tree.Node {
	return m.t.NodeAtCurrentOffset()
}

func (m *Model) GetCurrNodeDesendantDiffs() []*gitdiff.File {
	var files []*gitdiff.File
	for _, node := range m.GetCurrNode().AllNodes() {
		if file, ok := node.GivenValue().(*filenode.FileNode); ok {
			files = append(files, file.File)
		}
	}
	return files
}

// SetCursorNoScroll updates the selected file without scrolling the viewport.
// Use this when the user clicks on a file they can already see.
func (m *Model) SetCursorNoScroll(cursor int) {
	if len(m.files) == 0 {
		return
	}
	scroll := m.t.ViewportYOffset()
	m.rebuildTree()

	m.t.SetYOffset(cursor)
	m.t.SetViewportYOffset(scroll)
}

func (m *Model) CurrNodePath() string {
	fullpath := ""
	switch val := m.t.NodeAtCurrentOffset().GivenValue().(type) {
	case *filenode.FileNode:
		fullpath = filenode.GetFileName(val.File)
	case *dirnode.DirNode:
		fullpath = val.FullPath
	}
	return fullpath
}

func (m *Model) CopyCurrNodePath() tea.Cmd {
	fullpath := m.CurrNodePath()
	err := clipboard.WriteAll(fullpath)
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
