package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	zone "github.com/lrstanley/bubblezone"

	"github.com/dlvhdr/diffnav/pkg/config"
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

	// Zone IDs for bubblezone click detection.
	zoneSearchBox     = "searchbox"
	zoneFileTree      = "filetree"
	zoneSearchResults = "searchresults"
	zoneDiffViewer    = "diffviewer"

	// Sidebar resize detection threshold in pixels.
	sidebarGrabThreshold = 2

	// Sidebar width constraints.
	sidebarMinWidth  = 20
	sidebarHideWidth = 10

	// Scroll speed in lines per wheel tick.
	scrollLines = 3
)

type Panel int

const (
	FileTreePanel Panel = iota
	DiffViewerPanel
)

type mainModel struct {
	input              string
	files              []*gitdiff.File
	cursor             int
	fileTree           filetree.Model
	diffViewer         diffviewer.Model
	width              int
	height             int
	isShowingFileTree  bool
	activePanel        Panel
	search             textinput.Model
	help               help.Model
	resultsVp          viewport.Model
	resultsCursor      int
	searching          bool
	filtered           []string
	config             config.Config
	draggingSidebar    bool
	customSidebarWidth int
	iconStyle          string
	sideBySide         bool
}

func New(input string, cfg config.Config) mainModel {
	m := mainModel{input: input, isShowingFileTree: cfg.UI.ShowFileTree, activePanel: FileTreePanel, config: cfg, iconStyle: cfg.UI.Icons, sideBySide: cfg.UI.SideBySide}
	m.fileTree = filetree.New(cfg.UI.Icons, cfg.UI.ColorFileNames)
	m.diffViewer = diffviewer.New(cfg.UI.SideBySide)

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
	m.search.Prompt = " "
	m.search.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	m.search.Placeholder = "Filter files 󰬛 "
	m.search.PlaceholderStyle = lipgloss.NewStyle().MaxWidth(lipgloss.Width(m.search.Placeholder)).Foreground(lipgloss.Color("8"))
	m.search.Width = cfg.UI.FileTreeWidth - 5

	m.resultsVp = viewport.Model{}

	return m
}

func (m mainModel) Init() tea.Cmd {
	return tea.Batch(tea.EnterAltScreen, m.fetchFileTree, m.diffViewer.Init())
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Handle mouse events regardless of search mode
	if msg, ok := msg.(tea.MouseMsg); ok {
		return m.handleMouse(msg)
	}

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

				m.resultsVp.Width = m.config.UI.SearchTreeWidth
				m.resultsVp.Height = m.height - m.footerHeight() - m.headerHeight() - searchHeight
				m.resultsVp.SetContent(m.resultsView())

				dfCmd := m.diffViewer.SetSize(m.width-m.sidebarWidth(), m.height-m.footerHeight()-m.headerHeight())
				cmds = append(cmds, dfCmd, m.search.Focus())
			case "e":
				m.isShowingFileTree = !m.isShowingFileTree
				if m.isShowingFileTree {
					m.customSidebarWidth = 0 // Reset to default width.
				} else {
					m.activePanel = DiffViewerPanel
				}
				dfCmd := m.diffViewer.SetSize(m.width-m.sidebarWidth(), m.height-m.footerHeight()-m.headerHeight())
				cmds = append(cmds, dfCmd)
			case "i":
				m.cycleIconStyle()
			case "s":
				m.sideBySide = !m.sideBySide
				cmd = m.diffViewer.SetSideBySide(m.sideBySide)
				cmds = append(cmds, cmd)
			case "tab":
				if m.isShowingFileTree {
					if m.activePanel == FileTreePanel {
						m.activePanel = DiffViewerPanel
					} else {
						m.activePanel = FileTreePanel
					}
				}
			case "up", "k", "ctrl+p":
				if m.activePanel == FileTreePanel {
					if m.cursor > 0 {
						m.diffViewer.GoToTop()
						cmd = m.setCursor(m.cursor - 1)
						cmds = append(cmds, cmd)
					}
				} else {
					m.diffViewer.LineUp(1)
				}
			case "down", "j", "ctrl+n":
				if m.activePanel == FileTreePanel {
					if m.cursor < len(m.files)-1 {
						m.diffViewer.GoToTop()
						cmd = m.setCursor(m.cursor + 1)
						cmds = append(cmds, cmd)
					}
				} else {
					m.diffViewer.LineDown(1)
				}
			case "y":
				cmd = m.fileTree.CopyFilePath(m.cursor)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			case "o":
				cmd = m.openInEditor()
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}

		case tea.WindowSizeMsg:
			m.help.Width = msg.Width
			m.width = msg.Width
			m.height = msg.Height
			dfCmd := m.diffViewer.SetSize(m.width-m.sidebarWidth(), m.height-m.footerHeight()-m.headerHeight())
			cmds = append(cmds, dfCmd)
			ftCmd := m.fileTree.SetSize(m.sidebarWidth(), m.height-m.footerHeight()-m.headerHeight()-searchHeight)
			cmds = append(cmds, ftCmd)

		case fileTreeMsg:
			m.files = msg.files
			if len(m.files) == 0 {
				return m, tea.Quit
			}
			m.fileTree = m.fileTree.SetFiles(m.files)
			cmd = m.setCursor(0)
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

	// Route messages: key messages go only to active panel, other messages go to both.
	// Exception: ctrl+d/ctrl+u always go to diffViewer for scrolling.
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+d", "ctrl+u":
			m.diffViewer, cmd = m.diffViewer.Update(msg)
			cmds = append(cmds, cmd)
		default:
			if m.activePanel == DiffViewerPanel {
				m.diffViewer, cmd = m.diffViewer.Update(msg)
				cmds = append(cmds, cmd)
			} else {
				m.fileTree, cmd = m.fileTree.Update(msg)
				cmds = append(cmds, cmd)
			}
		}
	default:
		m.diffViewer, cmd = m.diffViewer.Update(msg)
		cmds = append(cmds, cmd)
		m.fileTree, cmd = m.fileTree.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *mainModel) cycleIconStyle() {
	switch m.iconStyle {
	case filenode.IconsASCII:
		m.iconStyle = filenode.IconsUnicode
	case filenode.IconsUnicode:
		m.iconStyle = filenode.IconsNerdFonts
	case filenode.IconsNerdFonts:
		m.iconStyle = filenode.IconsNerdFontsAlt
	default:
		m.iconStyle = filenode.IconsASCII
	}
	m.fileTree = m.fileTree.SetIconStyle(m.iconStyle)
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
				dfCmd := m.diffViewer.SetSize(m.width-m.sidebarWidth(), m.height-m.footerHeight()-m.headerHeight())
				cmds = append(cmds, dfCmd)
			case "ctrl+c":
				return m, []tea.Cmd{tea.Quit}
			case "enter":
				m.stopSearch()
				dfCmd := m.diffViewer.SetSize(m.width-m.sidebarWidth(), m.height-m.footerHeight()-m.headerHeight())
				cmds = append(cmds, dfCmd)

				selected := m.filtered[m.resultsCursor]
				for i, f := range m.files {
					if filenode.GetFileName(f) == selected {
						m.cursor = i
						m.diffViewer, cmd = m.diffViewer.SetFilePatch(f)
						m.fileTree = m.fileTree.SetCursor(i)
						cmds = append(cmds, cmd)
						break
					}
				}

			case "ctrl+n", "down":
				m.resultsCursor = min(len(m.filtered)-1, m.resultsCursor+1)
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
	// Determine colors based on active panel.
	leftColor := lipgloss.Color("8")
	rightColor := lipgloss.Color("8")
	if m.activePanel == FileTreePanel && !m.searching {
		leftColor = lipgloss.Color("4")
	} else if m.activePanel == DiffViewerPanel {
		rightColor = lipgloss.Color("4")
	}

	// Build T-shaped separator line.
	separator := ""
	if m.width > 0 {
		if m.isShowingFileTree {
			sidebarW := m.sidebarWidth()
			rightW := m.width - sidebarW - 1
			if rightW < 0 {
				rightW = 0
			}
			leftLine := lipgloss.NewStyle().Foreground(leftColor).Render(strings.Repeat("─", sidebarW))
			junction := lipgloss.NewStyle().Foreground(leftColor).Render("┬")
			rightLine := lipgloss.NewStyle().Foreground(rightColor).Render(strings.Repeat("─", rightW))
			separator = leftLine + junction + rightLine
		} else {
			separator = lipgloss.NewStyle().Foreground(rightColor).Render(strings.Repeat("─", m.width))
		}
	}

	sidebar := ""
	if m.isShowingFileTree {
		searchBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			MaxHeight(3).
			Width(m.sidebarWidth() - 2).
			Render(m.search.View())
		searchBox = zone.Mark(zoneSearchBox, searchBox)

		content := ""
		width := m.sidebarWidth()
		if m.searching {
			content = zone.Mark(zoneSearchResults, m.resultsVp.View())
		} else {
			content = zone.Mark(zoneFileTree, m.fileTree.View())
		}

		content = lipgloss.NewStyle().
			Width(width).
			Height(m.height - m.footerHeight() - m.headerHeight() - 1).Render(lipgloss.JoinVertical(lipgloss.Left, searchBox, content))

		sidebar = lipgloss.NewStyle().
			Width(width).
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(leftColor).Render(content)
	} else {
		// Show a thin grab line when sidebar is hidden.
		// Width(0) means only the border is rendered (1 char).
		grabLine := lipgloss.NewStyle().
			Width(0).
			Height(m.height-m.footerHeight()-m.headerHeight()-1).
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(lipgloss.Color("8")).
			Render("")
		sidebar = grabLine
	}
	dv := lipgloss.NewStyle().MaxHeight(m.height - m.footerHeight() - m.headerHeight() - 1).Width(m.width - m.sidebarWidth()).Render(m.diffViewer.View())
	dv = zone.Mark(zoneDiffViewer, dv)

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, dv)

	var sections []string

	if !m.config.UI.HideHeader {
		header := lipgloss.NewStyle().Width(m.width).
			Foreground(lipgloss.Color("6")).
			Bold(true).
			Render("DIFFNAV")
		sections = append(sections, header)
	}

	sections = append(sections, separator)
	sections = append(sections, mainContent)

	if !m.config.UI.HideFooter {
		sections = append(sections, m.footerView())
	}

	return zone.Scan(lipgloss.JoinVertical(lipgloss.Left, sections...))
}

type fileTreeMsg struct {
	files []*gitdiff.File
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
		fName := utils.TruncateString(" "+f, m.config.UI.SearchTreeWidth-2)
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
		return m.config.UI.SearchTreeWidth
	} else if m.isShowingFileTree {
		if m.customSidebarWidth > 0 {
			return m.customSidebarWidth
		}
		return m.config.UI.FileTreeWidth
	}
	return 0
}

func (m mainModel) headerHeight() int {
	if m.config.UI.HideHeader {
		return 0
	}
	return headerHeight
}

func (m mainModel) footerHeight() int {
	if m.config.UI.HideFooter {
		return 0
	}
	return footerHeight
}

func (m *mainModel) stopSearch() {
	m.searching = false
	m.search.SetValue("")
	m.search.Blur()
	m.search.Width = m.sidebarWidth() - 5
}

func (m *mainModel) setCursor(cursor int) tea.Cmd {
	var cmd tea.Cmd
	m.cursor = cursor
	m.diffViewer, cmd = m.diffViewer.SetFilePatch(m.files[m.cursor])
	m.fileTree = m.fileTree.SetCursor(m.cursor)
	return cmd
}

func (m mainModel) openInEditor() tea.Cmd {
	if len(m.files) == 0 {
		return nil
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		return nil
	}

	filePath := filenode.GetFileName(m.files[m.cursor])
	c := exec.Command(editor, filePath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return nil
	})
}

func (m mainModel) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Handle scroll wheel first.
	if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
		return m.handleScroll(msg)
	}

	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button == tea.MouseButtonLeft {
			// Keep coordinate check for resize border (hybrid approach).
			sidebarWidth := m.sidebarWidth()
			if m.isShowingFileTree && abs(msg.X-sidebarWidth) <= sidebarGrabThreshold {
				m.draggingSidebar = true
				return m, nil
			}
			// Allow grabbing the line when sidebar is hidden.
			if !m.isShowingFileTree && msg.X <= sidebarGrabThreshold {
				m.draggingSidebar = true
				m.isShowingFileTree = true
				return m, nil
			}

			// Zone-based detection for everything else.
			if zone.Get(zoneSearchBox).InBounds(msg) {
				return m.handleSearchBoxClick()
			}
			if m.searching && zone.Get(zoneSearchResults).InBounds(msg) {
				return m.handleSearchResultClick(msg)
			}
			if !m.searching && zone.Get(zoneFileTree).InBounds(msg) {
				return m.handleFileTreeClick(msg)
			}
		}

	case tea.MouseActionRelease:
		m.draggingSidebar = false

	case tea.MouseActionMotion:
		if m.draggingSidebar {
			return m.handleSidebarDrag(msg)
		}
	}

	return m, nil
}

func (m mainModel) handleSearchResultClick(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Use zone-relative coordinates.
	_, y := zone.Get(zoneSearchResults).Pos(msg)
	if y < 0 {
		return m, nil
	}
	clickedIndex := y + m.resultsVp.YOffset
	if clickedIndex >= len(m.filtered) {
		return m, nil
	}

	// Select the clicked result.
	selected := m.filtered[clickedIndex]
	m.stopSearch()

	var cmd tea.Cmd
	var cmds []tea.Cmd
	dfCmd := m.diffViewer.SetSize(m.width-m.sidebarWidth(), m.height-footerHeight-headerHeight)
	cmds = append(cmds, dfCmd)

	for i, f := range m.files {
		if filenode.GetFileName(f) == selected {
			m.cursor = i
			m.diffViewer, cmd = m.diffViewer.SetFilePatch(f)
			m.fileTree = m.fileTree.SetCursor(i)
			cmds = append(cmds, cmd)
			break
		}
	}

	return m, tea.Batch(cmds...)
}

func (m mainModel) handleSearchBoxClick() (tea.Model, tea.Cmd) {
	if m.searching {
		return m, nil
	}
	m.searching = true
	m.search.Width = m.sidebarWidth() - 5
	m.search.SetValue("")
	m.resultsCursor = 0
	m.filtered = make([]string, 0)

	m.resultsVp.Width = m.config.UI.SearchTreeWidth
	m.resultsVp.Height = m.height - footerHeight - headerHeight - searchHeight
	m.resultsVp.SetContent(m.resultsView())

	dfCmd := m.diffViewer.SetSize(m.width-m.sidebarWidth(), m.height-footerHeight-headerHeight)
	return m, tea.Batch(dfCmd, m.search.Focus())
}

func (m mainModel) handleFileTreeClick(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Use zone-relative coordinates.
	_, y := zone.Get(zoneFileTree).Pos(msg)
	if y < 0 {
		return m, nil
	}
	clickedY := y + m.fileTree.GetYOffset()

	// Find file at this Y position using tree traversal.
	filePath := m.fileTree.GetFileAtY(clickedY)
	if filePath == "" {
		return m, nil
	}

	// Find file index by path.
	for i, f := range m.files {
		if filenode.GetFileName(f) == filePath {
			m.cursor = i
			m.diffViewer.GoToTop()
			var cmd tea.Cmd
			m.diffViewer, cmd = m.diffViewer.SetFilePatch(f)
			// Use SetCursorNoScroll to avoid jumping the file tree view.
			m.fileTree = m.fileTree.SetCursorNoScroll(i)
			return m, cmd
		}
	}
	return m, nil
}

func (m mainModel) handleScroll(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	lines := scrollLines

	// Check if scrolling in sidebar (file tree or search results).
	if zone.Get(zoneFileTree).InBounds(msg) || zone.Get(zoneSearchResults).InBounds(msg) {
		if msg.Button == tea.MouseButtonWheelUp {
			if m.searching {
				m.resultsVp.LineUp(lines)
			} else {
				m.fileTree.ScrollUp(lines)
			}
		} else {
			if m.searching {
				m.resultsVp.LineDown(lines)
			} else {
				m.fileTree.ScrollDown(lines)
			}
		}
		return m, nil
	}

	// Check if scrolling in diff viewer.
	if zone.Get(zoneDiffViewer).InBounds(msg) {
		if msg.Button == tea.MouseButtonWheelUp {
			m.diffViewer.ScrollUp(lines)
		} else {
			m.diffViewer.ScrollDown(lines)
		}
	}
	return m, nil
}

func (m mainModel) handleSidebarDrag(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Hide sidebar if dragged below threshold.
	if msg.X < sidebarHideWidth {
		m.isShowingFileTree = false
		m.draggingSidebar = false
		cmd := m.diffViewer.SetSize(m.width, m.height-footerHeight-headerHeight)
		return m, cmd
	}

	// Clamp to reasonable bounds.
	minWidth := sidebarMinWidth
	maxWidth := m.width / 2
	newWidth := max(minWidth, min(maxWidth, msg.X))

	m.customSidebarWidth = newWidth

	// Resize components.
	cmds := []tea.Cmd{}
	cmds = append(cmds, m.diffViewer.SetSize(m.width-m.sidebarWidth(), m.height-footerHeight-headerHeight))
	cmds = append(cmds, m.fileTree.SetSize(m.sidebarWidth(), m.height-footerHeight-headerHeight-searchHeight))

	return m, tea.Batch(cmds...)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
