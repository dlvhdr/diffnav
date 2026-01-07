package filetree

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"

	"github.com/dlvhdr/diffnav/pkg/constants"
	"github.com/dlvhdr/diffnav/pkg/filenode"
	"github.com/dlvhdr/diffnav/pkg/ui/common"
	"github.com/dlvhdr/diffnav/pkg/utils"
)

type Model struct {
	files        []*gitdiff.File
	tree         *tree.Tree
	vp           viewport.Model
	selectedFile *string
}

// isRootHidden returns true if the tree root is hidden (not displayed).
func (m Model) isRootHidden() bool {
	return m.tree != nil && m.tree.Value() == dirIcon+"."
}

func (m Model) SetFiles(files []*gitdiff.File) Model {
	m.files = files
	t := buildFullFileTree(files)
	collapsed := collapseTree(t)
	m.tree, _ = truncateTree(collapsed, 0, 0, 0)
	m.vp.SetContent(m.printWithoutRoot())
	return m
}

func (m Model) SetCursor(cursor int) Model {
	if len(m.files) == 0 {
		return m
	}
	name := filenode.GetFileName(m.files[cursor])
	m.selectedFile = &name
	applyStyles(m.tree, m.selectedFile)
	m.scrollSelectedFileIntoView(m.tree)
	m.vp.SetContent(m.printWithoutRoot())
	return m
}

// SetCursorNoScroll updates the selected file without scrolling the viewport.
// Use this when the user clicks on a file they can already see.
func (m Model) SetCursorNoScroll(cursor int) Model {
	if len(m.files) == 0 {
		return m
	}
	name := filenode.GetFileName(m.files[cursor])
	m.selectedFile = &name
	applyStyles(m.tree, m.selectedFile)
	m.vp.SetContent(m.printWithoutRoot())
	return m
}

func (m Model) CopyFilePath(cursor int) tea.Cmd {
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

const contextLines = 15

func (m *Model) scrollSelectedFileIntoView(t *tree.Tree) {
	children := t.Children()
	found := false
	for i := 0; i < children.Length(); i++ {
		child := children.At(i)
		switch child := child.(type) {
		case *tree.Tree:
			m.scrollSelectedFileIntoView(child)
		case filenode.FileNode:
			if child.Path() == *m.selectedFile {
				// offset is 1-based, so we need to subtract 1
				offset := child.YOffset - 1 - contextLines
				if m.isRootHidden() {
					offset = offset - 1
				}
				m.vp.SetYOffset(offset)
				found = true
				break
			}
		}
		if found {
			break
		}
	}
}

func New() Model {
	return Model{
		files: []*gitdiff.File{},
		vp:    viewport.Model{},
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	// Ignore navigation keys - they're handled by mainModel for file selection.
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "down", "j", "k", "ctrl+p", "ctrl+n":
			return m, nil
		}
	}
	m.vp, _ = m.vp.Update(msg)
	return m, nil
}

var indenter = func(children tree.Children, index int) string {
	if children.Length()-1 == index {
		return " "
	}
	return "│"
}

var enumerator = func(children tree.Children, index int) string {
	if children.Length()-1 == index {
		return "╰"
	}
	return "├"
}

func (m Model) View() string {
	return m.vp.View()
}

// SetSize implements the Component interface.
func (m *Model) SetSize(width, height int) tea.Cmd {
	m.vp.Width = width
	m.vp.Height = height
	return nil
}

// GetYOffset returns the viewport's current Y scroll offset.
func (m Model) GetYOffset() int {
	return m.vp.YOffset
}

// GetFileAtY returns the file path at the given Y coordinate (0-indexed visual line), or "" if none.
func (m Model) GetFileAtY(y int) string {
	if m.tree == nil {
		return ""
	}
	// Convert visual line (0-indexed) to YOffset (1-indexed from tree traversal).
	// YOffset starts at 1 for root, 2 for first child, etc.
	yOffset := y + 1
	if m.isRootHidden() {
		yOffset++ // Root is hidden, so visual line 0 = YOffset 2.
	}
	return m.findFileAtY(m.tree, yOffset)
}

// findFileAtY traverses the tree to find the FileNode at position y.
func (m Model) findFileAtY(t *tree.Tree, y int) string {
	children := t.Children()
	for i := 0; i < children.Length(); i++ {
		child := children.At(i)
		switch c := child.(type) {
		case *tree.Tree:
			if result := m.findFileAtY(c, y); result != "" {
				return result
			}
		case filenode.FileNode:
			if c.YOffset == y {
				return c.Path()
			}
		}
	}
	return ""
}

// ScrollUp scrolls the viewport up by the given number of lines.
func (m *Model) ScrollUp(lines int) {
	m.vp.LineUp(lines)
}

// ScrollDown scrolls the viewport down by the given number of lines.
func (m *Model) ScrollDown(lines int) {
	m.vp.LineDown(lines)
}

func (m Model) printWithoutRoot() string {
	if !m.isRootHidden() {
		return m.tree.String()
	}

	s := ""
	children := m.tree.Children()
	for i := 0; i < children.Length(); i++ {
		child := children.At(i)
		switch child := child.(type) {
		case *tree.Tree:
			normalized := normalizeDepth(child, 0)
			applyStyles(normalized, m.selectedFile)

			s += normalized.String()
		case filenode.FileNode:
			child.Depth = 0
			s += applyStyleToNode(child, m.selectedFile).Render(child.Value())
		}
		if i < children.Length()-1 {
			s += "\n"
		}
	}
	return s
}

func normalizeDepth(node *tree.Tree, depth int) *tree.Tree {
	t := tree.Root(node.Value())
	children := node.Children()
	for i := 0; i < children.Length(); i++ {
		child := children.At(i)
		switch child := child.(type) {
		case *tree.Tree:
			sub := normalizeDepth(child, depth+1)
			t.Child(sub)
		case filenode.FileNode:
			child.Depth = depth + 1
			t.Child(child)
		}
	}
	return t
}

func buildFullFileTree(files []*gitdiff.File) *tree.Tree {
	t := tree.Root(".")
	for _, file := range files {
		subTree := t

		name := filenode.GetFileName(file)
		dir := filepath.Dir(name)
		parts := strings.Split(dir, string(os.PathSeparator))
		path := ""

		// walk the tree to find existing path
		for _, part := range parts {
			found := false
			for j := 0; j < subTree.Children().Length(); j++ {
				child := subTree.Children().At(j)
				if child.Value() == part {
					switch child := child.(type) {
					case *tree.Tree:
						subTree = child
					}
					path = path + part + string(os.PathSeparator)
					found = true
					break
				}
			}
			if !found {
				break
			}
		}

		// path does not exist from this point, need to creat it
		leftover := strings.TrimPrefix(name, path)
		parts = strings.Split(leftover, string(os.PathSeparator))
		for i, part := range parts {
			var c *tree.Tree
			if i == len(parts)-1 {
				subTree.Child(filenode.FileNode{File: file})
			} else {
				c = tree.Root(part)
				subTree.Child(c)
				subTree = c
			}
		}
	}

	return t
}

func collapseTree(t *tree.Tree) *tree.Tree {
	children := t.Children()
	newT := tree.Root(t.Value())
	if children.Length() == 0 {
		return newT
	}

	for i := 0; i < children.Length(); i++ {
		child := t.Children().At(i)
		switch child := child.(type) {
		case *tree.Tree:
			collapsedChild := collapseTree(child)
			newT.Child(collapsedChild)
		default:
			newT.Child(child)
		}
	}

	newChildren := newT.Children()
	if newChildren.Length() == 1 {
		child := newChildren.At(0)
		switch child := child.(type) {
		case *tree.Tree:
			if t.Value() == "." {
				return child
			}

			val := t.Value() + string(os.PathSeparator) + child.Value()
			collapsed := tree.Root(val).Child(child.Children())
			return collapsed
		}
	}

	return newT
}

const dirIcon = " "

func truncateTree(t *tree.Tree, depth int, numNodes int, numChildren int) (*tree.Tree, int) {
	newT := tree.Root(utils.TruncateString(dirIcon+t.Value(), constants.OpenFileTreeWidth-depth*2))
	numNodes++
	children := t.Children()
	for i := 0; i < children.Length(); i++ {
		child := children.At(i)
		numChildren++
		switch child := child.(type) {
		case *tree.Tree:
			sub, subNum := truncateTree(child, depth+1, numNodes, 0)
			numChildren += subNum
			numNodes += subNum + 1
			newT.Child(sub)
		case filenode.FileNode:
			numNodes++
			newT.Child(filenode.FileNode{File: child.File, Depth: depth + 1, YOffset: numNodes})
		default:
			newT.Child(child)
		}
	}
	return newT, numChildren
}

func applyStyles(t *tree.Tree, selectedFile *string) {
	enumeratorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).PaddingRight(1)
	rootStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	t.Enumerator(enumerator).Indenter(indenter).
		EnumeratorStyle(enumeratorStyle).
		ItemStyleFunc(applyStyle(selectedFile)).RootStyle(rootStyle)
}

func applyStyle(selectedFile *string) tree.StyleFunc {
	return func(children tree.Children, i int) lipgloss.Style {
		return applyStyleAux(children, i, selectedFile)
	}
}

func applyStyleAux(children tree.Children, i int, selectedFile *string) lipgloss.Style {
	st := lipgloss.NewStyle()
	if children.Length() == 0 {
		return st
	}
	child := children.At(i)
	return applyStyleToNode(child, selectedFile)
}

func applyStyleToNode(node tree.Node, selectedFile *string) lipgloss.Style {
	st := lipgloss.NewStyle()
	switch n := node.(type) {
	case filenode.FileNode:
		if selectedFile != nil && n.Path() == *selectedFile {
			return st.Bold(true).Foreground(lipgloss.Color("5")).Reverse(true)
		}
		return st
	case *tree.Tree:
		return st.Foreground(lipgloss.Color("4"))
	default:
		return st
	}
	return st
}
