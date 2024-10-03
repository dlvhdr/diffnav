package ui

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"

	"github.com/dlvhdr/diffnav/pkg/constants"
	"github.com/dlvhdr/diffnav/pkg/filenode"
	"github.com/dlvhdr/diffnav/pkg/ui/common"
	"github.com/dlvhdr/diffnav/pkg/ui/panes/diffviewer"
	"github.com/dlvhdr/diffnav/pkg/ui/panes/filetree"
	"github.com/dlvhdr/diffnav/pkg/utils"
)

const (
	footerHeight = 2
	headerHeight = 2
	searchHeight = 3
)

type mainModel struct {
	input             string
	files             []*gitdiff.File
	cursor            int
	fileTree          filetree.Model
	diffViewer        diffviewer.Model
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

func New(input string) mainModel {
	m := mainModel{input: input, isShowingFileTree: true}
	m.fileTree = filetree.New()
	m.diffViewer = diffviewer.New()

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
	m.search.Placeholder = "Filter files 󰬛 "
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

	if !m.searching {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "t":
				m.searching = true
				m.search.Width = m.sidebarWidth() - 5
				m.search.SetValue("")
				m.resultsCursor = 0
				m.filtered = make([]string, 0)

				m.resultsVp.Width = constants.SearchingFileTreeWidth
				m.resultsVp.Height = m.height - footerHeight - headerHeight - searchHeight
				m.resultsVp.SetContent(m.resultsView())

				dfCmd := m.diffViewer.SetSize(m.width-m.sidebarWidth(), m.height-footerHeight-headerHeight)
				cmds = append(cmds, dfCmd, m.search.Focus())
			case "e":
				m.isShowingFileTree = !m.isShowingFileTree
				dfCmd := m.diffViewer.SetSize(m.width-m.sidebarWidth(), m.height-footerHeight-headerHeight)
				cmds = append(cmds, dfCmd)
			case "up", "k", "ctrl+p":
				if m.cursor > 0 {
					m.cursor--
					m.diffViewer, cmd = m.diffViewer.SetFilePatch(m.files[m.cursor])
					cmds = append(cmds, cmd)
				}
			case "down", "j", "ctrl+n":
				if m.cursor < len(m.files)-1 {
					m.cursor++
					m.diffViewer, cmd = m.diffViewer.SetFilePatch(m.files[m.cursor])
					cmds = append(cmds, cmd)
				}
			}

		case tea.WindowSizeMsg:
			m.help.Width = msg.Width
			m.width = msg.Width
			m.height = msg.Height
			dfCmd := m.diffViewer.SetSize(m.width-m.sidebarWidth(), m.height-footerHeight-headerHeight)
			cmds = append(cmds, dfCmd)
			ftCmd := m.fileTree.SetSize(m.sidebarWidth(), m.height-footerHeight-headerHeight-searchHeight)
			cmds = append(cmds, ftCmd)

		case fileTreeMsg:
			m.files = msg.files
			if len(m.files) == 0 {
				return m, tea.Quit
			}
			m.fileTree = m.fileTree.SetFiles(m.files)
			m.diffViewer, cmd = m.diffViewer.SetFilePatch(m.files[0])
			cmds = append(cmds, cmd)

		case common.ErrMsg:
			fmt.Printf("Error: %v\n", msg.Err)
			log.Fatal(msg.Err)
		}
	} else {
		var sCmds []tea.Cmd
		m, sCmds = m.searchUpdate(msg)
		cmds = append(cmds, sCmds...)
	}

	m.fileTree = m.fileTree.SetCursor(m.cursor)

	m.diffViewer, cmd = m.diffViewer.Update(msg)
	cmds = append(cmds, cmd)

	m.fileTree, cmd = m.fileTree.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m mainModel) searchUpdate(msg tea.Msg) (mainModel, []tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	if m.search.Focused() {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.stopSearch()
				dfCmd := m.diffViewer.SetSize(m.width-m.sidebarWidth(), m.height-footerHeight-headerHeight)
				cmds = append(cmds, dfCmd)
			case "ctrl+c":
				return m, []tea.Cmd{tea.Quit}
			case "enter":
				m.stopSearch()
				dfCmd := m.diffViewer.SetSize(m.width-m.sidebarWidth(), m.height-footerHeight-headerHeight)
				cmds = append(cmds, dfCmd)

				selected := m.filtered[m.resultsCursor]
				for i, f := range m.files {
					if filenode.GetFileName(f) == selected {
						m.cursor = i
						m.diffViewer, cmd = m.diffViewer.SetFilePatch(f)
						cmds = append(cmds, cmd)
						break
					}
				}

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
			if strings.Contains(strings.ToLower(filenode.GetFileName(f)), strings.ToLower(m.search.Value())) {
				filtered = append(filtered, filenode.GetFileName(f))
			}
		}
		m.filtered = filtered
		m.resultsVp.SetContent(m.resultsView())
	}

	return m, cmds
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
			Width(width).
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(lipgloss.Color("8")).Render(content)
	}
	dv := lipgloss.NewStyle().MaxHeight(m.height - footerHeight - headerHeight).Width(m.width - m.sidebarWidth()).Render(m.diffViewer.View())
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		lipgloss.JoinHorizontal(lipgloss.Top, sidebar, dv),
		footer,
	)
}

func (m mainModel) fetchFileTree() tea.Msg {
	// TODO: handle error
	files, _, err := gitdiff.Parse(strings.NewReader(m.input + "\n"))
	if err != nil {
		return common.ErrMsg{Err: err}
	}
	sortFiles(files)

	return fileTreeMsg{files: files}
}

type fileTreeMsg struct {
	files []*gitdiff.File
}

func sortFiles(files []*gitdiff.File) {
	slices.SortFunc(files, func(a *gitdiff.File, b *gitdiff.File) int {
		nameA := filenode.GetFileName(a)
		nameB := filenode.GetFileName(b)
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

func (m mainModel) footerView() string {
	return lipgloss.NewStyle().
		Width(m.width).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(lipgloss.Color("8")).
		Height(1).
		Render(m.help.ShortHelpView(getKeys()))

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

func (m mainModel) setDiffViewerDimensions() tea.Cmd {
	dfCmd := m.diffViewer.SetSize(m.width-m.sidebarWidth(), m.height-footerHeight-headerHeight)
	return dfCmd
}

func (m *mainModel) stopSearch() {
	m.searching = false
	m.search.SetValue("")
	m.search.Blur()
	m.search.Width = m.sidebarWidth() - 5
}
