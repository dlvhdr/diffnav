package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/dlvhdr/diffnav/pkg/constants"
)

type mainModel struct {
	input             string
	files             []*gitdiff.File
	cursor            int
	fileTree          tea.Model
	diffViewer        tea.Model
	width             int
	height            int
	isShowingFileTree bool
	help              help.Model
}

func newModel(input string) mainModel {
	m := mainModel{input: input, isShowingFileTree: true}
	m.fileTree = initialFileTreeModel()
	m.diffViewer = initialDiffModel()
	m.help = help.New()
	helpSt := lipgloss.NewStyle().Background(lipgloss.Color("233"))
	m.help.Styles.ShortKey = helpSt
	m.help.Styles.ShortDesc = helpSt
	m.help.Styles.ShortSeparator = helpSt
	m.help.Styles.FullKey = helpSt.Foreground(lipgloss.Color("254"))
	m.help.Styles.FullDesc = helpSt
	m.help.Styles.FullSeparator = helpSt
	m.help.Styles.Ellipsis = helpSt
	return m
}

func (m mainModel) Init() tea.Cmd {
	return tea.Batch(tea.EnterAltScreen, m.fetchFileTree, m.diffViewer.Init())
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "e":
			m.isShowingFileTree = !m.isShowingFileTree
			df, dfCmd := m.diffViewer.(diffModel).Update(dimensionsMsg{Width: m.width - m.getFileTreeWidth(), Height: m.height - footerHeight - headerHeight})
			m.diffViewer = df
			return m, dfCmd
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.diffViewer, cmd = m.diffViewer.(diffModel).SetFilePatch(m.files[m.cursor])
				cmds = append(cmds, cmd)
			}
		case "down", "j":
			if m.cursor < len(m.files)-1 {
				m.cursor++
				m.diffViewer, cmd = m.diffViewer.(diffModel).SetFilePatch(m.files[m.cursor])
				cmds = append(cmds, cmd)
			}
		}

	case tea.MouseMsg:

	case tea.WindowSizeMsg:
		m.help.Width = msg.Width
		m.width = msg.Width
		m.height = msg.Height
		df, dfCmd := m.diffViewer.(diffModel).Update(dimensionsMsg{Width: m.width - m.getFileTreeWidth(), Height: m.height - footerHeight - headerHeight})
		m.diffViewer = df
		cmds = append(cmds, dfCmd)
		ft, ftCmd := m.fileTree.(ftModel).Update(dimensionsMsg{Width: constants.OpenFileTreeWidth, Height: m.height - footerHeight - headerHeight})
		m.fileTree = ft
		cmds = append(cmds, ftCmd)

	case fileTreeMsg:
		m.files = msg.files
		if len(m.files) == 0 {
			return m, tea.Quit
		}
		m.fileTree = m.fileTree.(ftModel).SetFiles(m.files)
		m.diffViewer, cmd = m.diffViewer.(diffModel).SetFilePatch(m.files[0])
		cmds = append(cmds, cmd)

	case errMsg:
		fmt.Printf("Error: %v\n", msg.err)
		log.Fatal(msg.err)
	}

	m.fileTree = m.fileTree.(ftModel).SetCursor(m.cursor)

	m.diffViewer, cmd = m.diffViewer.Update(msg)
	cmds = append(cmds, cmd)
	m.fileTree, cmd = m.fileTree.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m mainModel) View() string {
	header := lipgloss.NewStyle().Width(m.width).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(lipgloss.Color("8")).
		Foreground(lipgloss.Color("2")).
		Render("󰊢 diffnav")
	footer := m.footerView()

	ft := ""
	ftWidth := m.getFileTreeWidth()
	if m.isShowingFileTree {
		ft = lipgloss.NewStyle().
			Width(constants.OpenFileTreeWidth).
			Height(m.height-footerHeight-headerHeight).
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(lipgloss.Color("8")).
			Render(m.fileTree.View())
	}
	dv := lipgloss.NewStyle().MaxHeight(m.height - footerHeight - headerHeight).Width(m.width - ftWidth).Render(m.diffViewer.View())
	content := lipgloss.JoinHorizontal(lipgloss.Top, ft, dv)
	return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}

func (m mainModel) getFileTreeWidth() int {
	if m.isShowingFileTree {
		return constants.OpenFileTreeWidth
	}

	return 0
}

type dimensionsMsg struct {
	Width  int
	Height int
}

func (m mainModel) fetchFileTree() tea.Msg {
	// TODO: handle error
	files, _, err := gitdiff.Parse(strings.NewReader(m.input + "\n"))
	if err != nil {
		return errMsg{err}
	}
	sortFiles(files)

	return fileTreeMsg{files: files}
}

type fileTreeMsg struct {
	files []*gitdiff.File
}

func sortFiles(files []*gitdiff.File) {
	slices.SortFunc(files, func(a *gitdiff.File, b *gitdiff.File) int {
		getName := func(f *gitdiff.File) string {
			if f.NewName != "" {
				return f.NewName
			}
			return f.OldName
		}
		nameA := getName(a)
		nameB := getName(b)
		dira := filepath.Dir(nameA)
		dirb := filepath.Dir(nameB)
		if dira != "." && dirb != "." && dira == dirb {
			return strings.Compare(strings.ToLower(nameA), strings.ToLower(nameB))
		}

		if dira != "." && dirb == "." {
			return -1
		}
		if dirb != "." && dira == "." {
			return 1
		}

		if dira != "." && dirb != "." {
			if strings.HasPrefix(dira, dirb) {
				return -1
			}

			if strings.HasPrefix(dirb, dira) {
				return 1
			}
		}

		return strings.Compare(strings.ToLower(nameA), strings.ToLower(nameB))
	})
}

const (
	footerHeight = 1
	headerHeight = 2
)

func (m mainModel) footerView() string {
	return lipgloss.NewStyle().
		Background(lipgloss.Color("233")).
		Width(m.width).
		Height(footerHeight).
		Render(m.help.FullHelpView(getKeys()))

}

func main() {
	stat, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}

	if stat.Mode()&os.ModeNamedPipe == 0 && stat.Size() == 0 {
		fmt.Println("Try piping in some text.")
		os.Exit(1)
	}

	reader := bufio.NewReader(os.Stdin)
	var b strings.Builder

	for {
		r, _, err := reader.ReadRune()
		if err != nil && err == io.EOF {
			break
		}
		_, err = b.WriteRune(r)
		if err != nil {
			fmt.Println("Error getting input:", err)
			os.Exit(1)
		}
	}

	if os.Getenv("DEBUG") == "true" {
		logger, _ := tea.LogToFile("debug.log", "debug")
		defer logger.Close()
	}

	input := ansi.Strip(b.String())
	p := tea.NewProgram(newModel(input), tea.WithMouseAllMotion())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
