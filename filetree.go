package main

import (
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
)

type ftModel struct {
	cursor int
	files  []string
}

func (m ftModel) SetFiles(files []string) ftModel {
	m.files = files
	return m
}

func (m ftModel) SetCursor(cursor int) ftModel {
	m.cursor = cursor
	return m
}

func initialFileTreeModel() ftModel {
	return ftModel{
		files: []string{},
	}
}

func (m ftModel) Init() tea.Cmd {
	return fetchFileTree
}

func (m ftModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m ftModel) File() string {
	if m.cursor < 0 || m.cursor >= len(m.files) {
		return ""
	}
	return m.files[m.cursor]
}

func (m ftModel) View() string {
	enumeratorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).PaddingRight(1)
	itemStyle := lipgloss.NewStyle().PaddingRight(1)
	rootStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true)

	s := ""
	root, err := os.Getwd()
	if err != nil {
		return err.Error()
	}
	t := tree.Root(filepath.Base(root)).
		EnumeratorStyle(enumeratorStyle).
		ItemStyle(itemStyle).
		RootStyle(rootStyle)
	for i, file := range m.files {

		// base := filepath.Base(file)
		// if m.cursor == i {
		// 	base = lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Render(base)
		// }

		// stat, _ := os.Stat(file)
		// if stat.IsDir() {
		// dir := filepath.Dir(file)
		// dirTree := tree.Root(dir).Child(base)
		// t.Child(dirTree)
		// } else {
		//   t.Child(file)
		// }

		if m.cursor == i {
			t = t.Child(lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Render(file))
		} else {
			t.Child(file)
		}
	}

	s += t.String()

	// for i, file := range m.files {
	// 	cursor := " "
	// 	if m.cursor == i {
	// 		cursor = ">"
	// 	}
	//
	// 	s += fmt.Sprintf("%s %s\n", cursor, file)
	// }

	return s
}

type errMsg struct {
	err error
}
