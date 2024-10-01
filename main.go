package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/x/ansi"

	"github.com/dlvhdr/diffnav/pkg/constants"
	filetree "github.com/dlvhdr/diffnav/pkg/file_tree"
	"github.com/dlvhdr/diffnav/pkg/utils"
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
	search            textinput.Model
	help              help.Model
	resultsVp         viewport.Model
	resultsCursor     int
	searching         bool
	filtered          []string
}

func newModel(input string) mainModel {
	m := mainModel{input: input, isShowingFileTree: true}
	m.fileTree = initialFileTreeModel()
	m.diffViewer = initialDiffModel()

	m.help = help.New()
	helpSt := lipgloss.NewStyle()
	m.help.ShortSeparator = " · "
	m.help.Styles.ShortKey = helpSt
	m.help.Styles.ShortDesc = helpSt
	m.help.Styles.ShortSeparator = helpSt
	m.help.Styles.ShortKey = helpSt.Foreground(lipgloss.Color("254"))
	m.help.Styles.ShortDesc = helpSt
	m.help.Styles.ShortSeparator = helpSt
	m.help.Styles.Ellipsis = helpSt

	m.search = textinput.New()
	m.search.ShowSuggestions = true
	m.search.KeyMap.AcceptSuggestion = key.NewBinding(key.WithKeys("tab"))
	m.search.Prompt = " "
	m.search.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	m.search.Placeholder = "Filter files 🅃"
	m.search.PlaceholderStyle = lipgloss.NewStyle().MaxWidth(lipgloss.Width(m.search.Placeholder)).Foreground(lipgloss.Color("8"))
	m.search.Width = constants.OpenFileTreeWidth - 5

	m.resultsVp = viewport.Model{}

	return m
}

func (m mainModel) Init() tea.Cmd {
	return tea.Batch(tea.EnterAltScreen, m.fetchFileTree, m.diffViewer.Init())
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	if m.search.Focused() {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				df, dfCmd := m.setDiffViewerDimensions()
				cmds = append(cmds, dfCmd)
				m.diffViewer = df
				m.search.SetValue("")
				m.search.Blur()
				m.searching = false
				break
			case "ctrl+c":
				return m, tea.Quit
			case "enter":
				m.searching = false
				m.search.SetValue("")
				m.search.Blur()
				selected := m.filtered[m.resultsCursor]
				df, dfCmd := m.setDiffViewerDimensions()
				cmds = append(cmds, dfCmd)
				m.diffViewer = df
				for i, f := range m.files {
					if filetree.GetFileName(f) == selected {
						m.cursor = i
						m.diffViewer, cmd = m.diffViewer.(diffModel).SetFilePatch(f)
						cmds = append(cmds, cmd)
						break
					}
				}

				return m, tea.Batch(cmds...)
			case "ctrl+n", "down":
				m.resultsCursor = min(len(m.files)-1, m.resultsCursor+1)
				m.resultsVp.LineDown(1)
			case "ctrl+p", "up":
				m.resultsCursor = max(0, m.resultsCursor-1)
				m.resultsVp.LineUp(1)
			default:
				m.resultsCursor = 0
			}
		}
		s, sc := m.search.Update(msg)
		cmds = append(cmds, sc)
		m.search = s
		filtered := make([]string, 0)
		for _, f := range m.files {
			if strings.Contains(strings.ToLower(filetree.GetFileName(f)), strings.ToLower(m.search.Value())) {
				filtered = append(filtered, filetree.GetFileName(f))
			}
		}
		m.filtered = filtered
		m.resultsVp.SetContent(m.resultsView())

		return m, tea.Batch(cmds...)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.searching {
			switch msg.String() {
			case "ctrl+n":
				if m.searching {
					m.resultsCursor = min(len(m.files)-1, m.resultsCursor+1)
					m.resultsVp.LineDown(1)
				}
			case "ctrl+p":
				if m.searching {
					m.resultsCursor = max(0, m.resultsCursor-1)
					m.resultsVp.LineUp(1)
				}
			}
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "t":
			m.searching = true
			m.search.Width = m.sidebarWidth() - 5
			m.search.SetValue("")
			m.resultsCursor = 0
			m.filtered = make([]string, 0)
			m.resultsVp.SetContent(m.resultsView())
			m.resultsVp.Height = m.height - footerHeight - headerHeight - searchHeight
			m.resultsVp.Width = constants.SearchingFileTreeWidth
			df, dfCmd := m.setDiffViewerDimensions()
			cmds = append(cmds, dfCmd)
			m.diffViewer = df
			cmds = append(cmds, m.search.Focus())
			return m, tea.Batch(cmds...)
		case "e":
			m.isShowingFileTree = !m.isShowingFileTree
			df, dfCmd := m.setDiffViewerDimensions()
			m.diffViewer = df
			return m, dfCmd
		case "up", "k", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
				m.diffViewer, cmd = m.diffViewer.(diffModel).SetFilePatch(m.files[m.cursor])
				cmds = append(cmds, cmd)
			}
		case "down", "j", "ctrl+n":
			if m.cursor < len(m.files)-1 {
				m.cursor++
				m.diffViewer, cmd = m.diffViewer.(diffModel).SetFilePatch(m.files[m.cursor])
				cmds = append(cmds, cmd)
			}
		}

	case tea.WindowSizeMsg:
		m.help.Width = msg.Width
		m.width = msg.Width
		m.height = msg.Height
		df, dfCmd := m.diffViewer.(diffModel).Update(dimensionsMsg{Width: m.width - m.sidebarWidth(), Height: m.height - footerHeight - headerHeight})
		m.diffViewer = df
		cmds = append(cmds, dfCmd)
		ft, ftCmd := m.fileTree.(ftModel).Update(dimensionsMsg{Width: m.sidebarWidth(), Height: m.height - footerHeight - headerHeight - searchHeight})
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

	s, sCmd := m.search.Update(msg)
	cmds = append(cmds, sCmd)
	m.search = s
	m.search.Width = m.sidebarWidth() - 5

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
		Foreground(lipgloss.Color("6")).
		Bold(true).
		Render("DIFFNAV")
	footer := m.footerView()

	sidebar := ""
	if m.isShowingFileTree {
		search := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			MaxHeight(3).
			Width(m.sidebarWidth() - 2).
			Render(m.search.View())

		content := ""
		width := m.sidebarWidth()
		if m.searching {
			content = m.resultsVp.View()
		} else {
			content = m.fileTree.View()
		}

		content = lipgloss.NewStyle().
			Width(width).
			Height(m.height - footerHeight - headerHeight).Render(lipgloss.JoinVertical(lipgloss.Left, search, content))

		sidebar = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(lipgloss.Color("8")).Render(content)
	}
	dv := lipgloss.NewStyle().MaxHeight(m.height - footerHeight - headerHeight).Width(m.width - m.sidebarWidth()).Render(m.diffViewer.View())
	content := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, dv)
	return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
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
		nameA := filetree.GetFileName(a)
		nameB := filetree.GetFileName(b)
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
	footerHeight = 2
	headerHeight = 2
	searchHeight = 3
)

func (m mainModel) footerView() string {
	return lipgloss.NewStyle().
		Width(m.width).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(lipgloss.Color("8")).
		Height(1).
		Render(m.help.ShortHelpView(getKeys()))

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

	var fileErr error
	logFile, fileErr := os.OpenFile("debug.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if fileErr == nil {
		log.SetOutput(logFile)
		log.SetTimeFormat(time.Kitchen)
		log.SetReportCaller(true)
		log.SetLevel(log.DebugLevel)
		defer logFile.Close()
		log.SetOutput(logFile)
		log.Debug("Starting diffnav, logging to debug.log")
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
	if strings.TrimSpace(input) == "" {
		fmt.Println("No input provided, exiting")
		return
	}
	p := tea.NewProgram(newModel(input), tea.WithMouseAllMotion())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

func (m mainModel) resultsView() string {
	sb := strings.Builder{}
	for i, f := range m.filtered {
		fName := utils.TruncateString(" "+f, constants.SearchingFileTreeWidth-2)
		if i == m.resultsCursor {
			sb.WriteString(lipgloss.NewStyle().Background(lipgloss.Color("#1b1b33")).Bold(true).Render(fName) + "\n")
		} else {
			sb.WriteString(fName + "\n")
		}
	}
	return sb.String()
}

func (m mainModel) sidebarWidth() int {
	if m.searching {
		return constants.SearchingFileTreeWidth
	} else if m.isShowingFileTree {
		return constants.OpenFileTreeWidth
	} else {
		return 0
	}
}

func (m mainModel) setDiffViewerDimensions() (tea.Model, tea.Cmd) {
	df, dfCmd := m.diffViewer.(diffModel).Update(dimensionsMsg{Width: m.width - m.sidebarWidth(), Height: m.height - footerHeight - headerHeight})
	return df, dfCmd
}
