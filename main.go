package main

import (
	"bufio"
	"bytes"
	"log"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type mainModel struct {
	files      []string
	cursor     int
	fileTree   tea.Model
	diffViewer tea.Model
	width      int
	height     int
}

func newModel() mainModel {
	m := mainModel{}
	m.fileTree = initialFileTreeModel()
	m.diffViewer = initialDiffModel()
	return m
}

func (m mainModel) Init() tea.Cmd {
	return tea.Batch(tea.EnterAltScreen, m.fileTree.Init(), m.diffViewer.Init())
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.diffViewer, cmd = m.diffViewer.(diffModel).SetFilePath(m.files[m.cursor])
				cmds = append(cmds, cmd)
			}
		case "down", "j":
			if m.cursor < len(m.files)-1 {
				m.cursor++
				m.diffViewer, cmd = m.diffViewer.(diffModel).SetFilePath(m.files[m.cursor])
				cmds = append(cmds, cmd)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case fileTreeMsg:
		m.files = msg.files
		m.fileTree = m.fileTree.(ftModel).SetFiles(m.files)
		m.diffViewer, cmd = m.diffViewer.(diffModel).SetFilePath(m.files[0])
		cmds = append(cmds, cmd)
	}

	m.fileTree = m.fileTree.(ftModel).SetCursor(m.cursor)

	m.diffViewer, cmd = m.diffViewer.Update(msg)
	cmds = append(cmds, cmd)
	m.fileTree, cmd = m.fileTree.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

const fileTreeWidth = 25

func (m mainModel) View() string {
	ft := lipgloss.NewStyle().
		Width(fileTreeWidth).
		Height(m.height).
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(lipgloss.Color("8")).
		Padding(0, 1).
		Render(m.fileTree.View())
	dv := lipgloss.NewStyle().MaxHeight(m.height).Width(m.width - fileTreeWidth).Render(m.diffViewer.View())
	return lipgloss.JoinHorizontal(lipgloss.Top, ft, dv)
}

func fetchFileTree() tea.Msg {
	c := exec.Command("git", "diff", "--name-only", "HEAD")
	stdout, err := c.Output()
	if err != nil {
		return errMsg{err: err}
	}
	scanner := bufio.NewScanner(bytes.NewReader(stdout))
	files := make([]string, 0)
	for scanner.Scan() {
		line := scanner.Text()
		files = append(files, line)
	}
	return fileTreeMsg{files: files}
}

type fileTreeMsg struct {
	files []string
}

func main() {
	p := tea.NewProgram(newModel())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
