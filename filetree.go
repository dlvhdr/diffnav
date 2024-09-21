package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
)

type ftModel struct {
	files        []*gitdiff.File
	tree         *tree.Tree
	selectedFile *string
}

func (m ftModel) SetFiles(files []*gitdiff.File) ftModel {
	m.files = files
	t := buildFullFileTree(files)
	log.Printf("full: %v\n", t)
	collapsed := collapseTree(t)
	log.Printf("collapsed: %v\n", collapsed)
	m.tree = truncateTree(collapsed, 0)
	return m
}

type FileNode struct {
	file  *gitdiff.File
	depth int
}

func (f FileNode) path() string {
	return getFileName(f.file)
}

func (f FileNode) Value() string {
	icon := " "
	status := " "
	if f.file.IsNew {
		status += lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render("")
	} else if f.file.IsDelete {
		status += lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("")
	} else {
		status += lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("")
	}

	depth := f.depth
	if f.depth > 0 {
		depth = depth - 1
	}
	depthWidth := depth * 2
	iconsWidth := lipgloss.Width(icon) + lipgloss.Width(status)
	nameMaxWidth := openFileTreeWidth - depthWidth - iconsWidth
	base := filepath.Base(f.path())
	name := truncateValue(base, nameMaxWidth)

	spacerWidth := openFileTreeWidth - lipgloss.Width(name) - iconsWidth - depthWidth
	if len(name) < len(base) {
		spacerWidth = spacerWidth - 1
	}
	spacer := ""
	if spacerWidth > 0 {
		spacer = strings.Repeat(" ", spacerWidth)
	}

	log.Printf("name: %s, nameWidth: %d, nameMaxWidth: %d, iconsWidth: %d, depthWidth: %d, spacerWidth: %d\n", name, lipgloss.Width(name), nameMaxWidth, iconsWidth, depth*2, spacerWidth)

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

func (m ftModel) SetCursor(cursor int) ftModel {
	if len(m.files) == 0 {
		return m
	}
	name := getFileName(m.files[cursor])
	m.selectedFile = &name
	applyStyles(m.tree, m.selectedFile)
	return m
}

func initialFileTreeModel() ftModel {
	return ftModel{
		files: []*gitdiff.File{},
	}
}

func (m ftModel) Init() tea.Cmd {
	return nil
}

func (m ftModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m ftModel) View() string {
	if m.tree == nil {
		return ""
	}

	return lipgloss.NewStyle().Width(openFileTreeWidth).MaxWidth(openFileTreeWidth).Render(m.printWithoutRoot())
}

type errMsg struct {
	err error
}

func (m ftModel) printWithoutRoot() string {
	if m.tree.Value() != "." {
		return m.tree.String()
	}

	s := ""
	children := m.tree.Children()
	for i := 0; i < children.Length(); i++ {
		child := children.At(i)
		switch child := child.(type) {
		case *tree.Tree:
			applyStyles(child, m.selectedFile)
			s += child.String()
		case FileNode:
			s += applyStyleToNode(child, m.selectedFile).Render(child.Value())
		}
		if i < children.Length()-1 {
			s += "\n"
		}
	}
	return s
}

func buildFullFileTree(files []*gitdiff.File) *tree.Tree {
	t := tree.Root(".")
	for _, file := range files {
		subTree := t

		name := getFileName(file)
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
				subTree.Child(FileNode{file: file})
			} else {
				c = tree.Root(part)
				subTree.Child(c)
				subTree = c
			}
		}
	}

	return t
}

func truncateValue(value string, width int) string {
	return TruncateString(value, width)
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

func truncateTree(t *tree.Tree, depth int) *tree.Tree {
	d := depth
	if d > 0 {
		d = d - 1
	}
	newT := tree.Root(truncateValue(t.Value(), openFileTreeWidth-d*2))
	children := t.Children()
	for i := 0; i < children.Length(); i++ {
		child := children.At(i)
		switch child := child.(type) {
		case *tree.Tree:
			newT.Child(truncateTree(child, depth+1))
		case FileNode:
			newT.Child(FileNode{file: child.file, depth: depth + 1})
		default:
			newT.Child(child)
		}
	}
	return newT
}

func applyStyles(t *tree.Tree, selectedFile *string) {
	enumeratorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).PaddingRight(1)
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
	st := lipgloss.NewStyle().Background(lipgloss.Color("2"))
	if children.Length() == 0 {
		return st
	}
	child := children.At(i)
	return applyStyleToNode(child, selectedFile)
}

func applyStyleToNode(node tree.Node, selectedFile *string) lipgloss.Style {
	st := lipgloss.NewStyle().MaxHeight(1)
	switch n := node.(type) {
	case FileNode:
		if selectedFile != nil && n.path() == *selectedFile {
			return st.Background(lipgloss.Color("#1b1b33")).Bold(true)
		}
	case *tree.Tree:
		return st.Foreground(lipgloss.Color("4"))
	default:
		return st
	}
	return st
}

func getFileName(file *gitdiff.File) string {
	if file.NewName != "" {
		return file.NewName
	}
	return file.OldName
}
