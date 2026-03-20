package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/tree"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/log/v2"
	"github.com/bluekeyes/go-gitdiff/gitdiff"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/dlvhdr/diffnav/pkg/config"
	"github.com/dlvhdr/diffnav/pkg/dirnode"
	"github.com/dlvhdr/diffnav/pkg/filenode"
	"github.com/dlvhdr/diffnav/pkg/ui/common"
	"github.com/dlvhdr/diffnav/pkg/ui/panes/diffviewer"
	"github.com/dlvhdr/diffnav/pkg/ui/panes/filetree"
	"github.com/dlvhdr/diffnav/pkg/ui/panes/help"
	"github.com/dlvhdr/diffnav/pkg/utils"
	"github.com/lrstanley/go-nf/glyphs/md"
	"github.com/lrstanley/go-nf/glyphs/neo"
)

const (
	minResizeStep = 6
	footerHeight  = 1
	headerHeight  = 2
	searchHeight  = 3

	// Zone IDs for bubblezone click detection.
	zoneSearchBox     = "searchbox"
	zoneFileTree      = "filetree"
	zoneSearchResults = "searchresults"
	zoneDiffViewer    = "diffviewer"
	zoneHelp          = "help"
	zoneHeader        = "header"

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
	input             string
	files             []*gitdiff.File
	fileTree          filetree.Model
	diffViewer        diffviewer.Model
	width             int
	height            int
	isShowingFileTree bool
	activePanel       Panel
	search            textinput.Model
	resultsVp         viewport.Model
	resultsCursor     int
	searching         bool
	filtered          []string
	config            config.Config
	draggingSidebar   bool
	iconStyle         string
	sideBySide        bool
	help              help.Model
	helpOpen          bool
	messageOpen       bool
	messageVp         viewport.Model
	preamble          string
	commitBranch      string
	cachedMeta        commitMeta
}

func New(input string, cfg config.Config) mainModel {
	m := mainModel{
		input: input, isShowingFileTree: cfg.UI.ShowFileTree,
		activePanel: FileTreePanel, config: cfg, iconStyle: cfg.UI.Icons, sideBySide: cfg.UI.SideBySide,
	}
	m.fileTree = filetree.New(cfg)
	m.fileTree.SetSize(cfg.UI.FileTreeWidth, 0)
	m.diffViewer = diffviewer.New(cfg.UI.SideBySide)
	m.help = help.New()
	m.help.SetKeys(KeyGroups())

	m.search = textinput.New()
	m.search.ShowSuggestions = true
	m.search.KeyMap.AcceptSuggestion = key.NewBinding(key.WithKeys("tab"))
	m.search.Prompt = " "
	m.search.Placeholder = "Filter files 󰬛 "
	m.search.SetStyles(textinput.Styles{
		Focused: textinput.StyleState{
			Placeholder: lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
			Prompt:      lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
		},
	})
	m.search.SetWidth(cfg.UI.FileTreeWidth - 2)

	m.resultsVp = viewport.Model{}

	return m
}

func (m mainModel) Init() tea.Cmd {
	return tea.Batch(m.fetchFileTree, m.diffViewer.Init())
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Handle mouse events regardless of search mode
	if msg, ok := msg.(tea.MouseMsg); ok {
		return m.handleMouse(msg)
	}

	if m.searching {
		var sCmds []tea.Cmd
		m, sCmds = m.searchUpdate(msg)
		cmds = append(cmds, sCmds...)
		return m, tea.Batch(cmds...)
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keys.ToggleHelp):
			m.helpOpen = !m.helpOpen
			m.messageOpen = false
			return m, tea.Batch(cmds...)
		case key.Matches(msg, keys.ToggleMessage):
			if m.preamble != "" {
				m.messageOpen = !m.messageOpen
				m.helpOpen = false
				if m.messageOpen {
					m.updateMessageVp()
					m.messageVp.GotoTop()
				}
			}
			return m, tea.Batch(cmds...)
		case (m.helpOpen || m.messageOpen) && (key.Matches(msg, keys.Quit) || msg.Key().Code == tea.KeyEscape):
			m.helpOpen = false
			m.messageOpen = false
			return m, tea.Batch(cmds...)
		case m.messageOpen && key.Matches(msg, keys.Down):
			m.messageVp.ScrollDown(1)
			return m, tea.Batch(cmds...)
		case m.messageOpen && key.Matches(msg, keys.CtrlD):
			m.messageVp.ScrollDown(m.messageVp.Height() / 2)
			return m, tea.Batch(cmds...)
		case m.messageOpen && key.Matches(msg, keys.Up):
			m.messageVp.ScrollUp(1)
			return m, tea.Batch(cmds...)
		case m.messageOpen && key.Matches(msg, keys.CtrlU):
			m.messageVp.ScrollUp(m.messageVp.Height() / 2)
			return m, tea.Batch(cmds...)
		case m.helpOpen || m.messageOpen:
			// Block all other keys while an overlay is open
			return m, tea.Batch(cmds...)
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Search):
			m.searching = true
			m.search.SetWidth(m.searchWidth())
			m.search.SetValue("")
			m.resultsCursor = 0
			m.setSearchResults()

			m.resultsVp.SetWidth(m.config.UI.SearchTreeWidth)
			m.resultsVp.SetHeight(m.mainContentHeight() - searchHeight)
			m.resultsVp.SetContent(m.resultsView())

			dfCmd := m.diffViewer.SetSize(m.width-m.sidebarWidth(), m.mainContentHeight())
			cmds = append(cmds, dfCmd, m.search.Focus())
		case key.Matches(msg, keys.ToggleFileTree):
			m.isShowingFileTree = !m.isShowingFileTree
			sidebarWidth := m.sidebarWidth()
			h := m.mainContentHeight()

			if !m.isShowingFileTree {
				m.activePanel = DiffViewerPanel
			} else {
				m.activePanel = FileTreePanel
			}
			treeWidth := sidebarWidth
			if sidebarWidth == 0 {
				treeWidth = m.config.UI.FileTreeWidth
			}

			m.fileTree.SetSize(treeWidth, h-searchHeight)
			m.search.SetWidth(m.searchWidth())
			dfCmd := m.diffViewer.SetSize(m.width-sidebarWidth, h)
			cmds = append(cmds, dfCmd)
		case key.Matches(msg, keys.ToggleIconStyle):
			m.cycleIconStyle()
		case key.Matches(msg, keys.ToggleDiffView):
			m.sideBySide = !m.sideBySide
			cmd = m.diffViewer.SetSideBySide(m.sideBySide)
			cmds = append(cmds, cmd)
		case key.Matches(msg, keys.SwitchPanel):
			if m.isShowingFileTree {
				if m.activePanel == FileTreePanel {
					m.activePanel = DiffViewerPanel
				} else {
					m.activePanel = FileTreePanel
				}
			}
		case key.Matches(msg, keys.PrevFile):
			m, cmd = m.moveToFile(-1)
			cmds = append(cmds, cmd)
		case key.Matches(msg, keys.NextFile):
			m, cmd = m.moveToFile(1)
			cmds = append(cmds, cmd)
		case key.Matches(msg, keys.Up):
			if m.activePanel == FileTreePanel {
				m, cmd = m.moveCursor(-1)
				cmds = append(cmds, cmd)
			} else {
				m.diffViewer.ScrollUp(1)
			}
		case key.Matches(msg, keys.Down):
			if m.activePanel == FileTreePanel {
				m, cmd = m.moveCursor(1)
				cmds = append(cmds, cmd)
			} else {
				m.diffViewer.ScrollDown(1)
			}
		case key.Matches(msg, keys.Copy):
			cmd = m.fileTree.CopyCurrNodePath()
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		case key.Matches(msg, keys.OpenInEditor):
			cmd = m.openInEditor()
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case tea.WindowSizeMsg:
		log.Info("got tea.WindowSizeMsg", "width", msg.Width, "height", msg.Height)
		m.help.Update(msg)
		m.width = msg.Width
		m.height = msg.Height
		dfCmd := m.diffViewer.SetSize(m.width-m.sidebarWidth(), m.mainContentHeight())
		cmds = append(cmds, dfCmd)

		tWidth, tHeight := m.sidebarWidth(), m.mainContentHeight()-searchHeight

		m.fileTree.SetSize(tWidth, tHeight)
		m.search.SetWidth(m.searchWidth())
		if m.messageOpen {
			m.updateMessageVp()
		}

	case fileTreeMsg:
		m.files = msg.files
		if len(m.files) == 0 {
			return m, tea.Quit
		}
		m.fileTree = m.fileTree.SetFiles(m.files)
		m.preamble = strings.TrimSpace(msg.preamble)
		m.commitBranch = msg.branch
		m.cachedMeta = m.parseCommitMeta()
		m.diffViewer.SetPreamble(m.preamble)
		m.diffViewer, cmd = m.diffViewer.SetDirPatch("/", m.fileTree.GetCurrNodeDesendantDiffs())
		cmds = append(cmds, cmd)

	case common.ErrMsg:
		fmt.Printf("Error: %v\n", msg.Err)
		log.Fatal(msg.Err)
	}

	// Route messages: key messages go only to active panel, other messages go to both.
	// Exception: ctrl+d/ctrl+u go to diffViewer for scrolling (unless an overlay is open).
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
				m.fileTree.Update(msg)
				cmds = append(cmds, cmd)
			}
		}
	default:
		m.diffViewer, cmd = m.diffViewer.Update(msg)
		cmds = append(cmds, cmd)
		m.fileTree.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *mainModel) mainContentHeight() int {
	return m.height - m.headerHeight() - m.footerHeight()
}

func (m *mainModel) cycleIconStyle() {
	switch m.iconStyle {
	case filenode.IconsASCII:
		m.iconStyle = filenode.IconsUnicode
	case filenode.IconsUnicode:
		m.iconStyle = filenode.IconsNerdStatus
	case filenode.IconsNerdStatus:
		m.iconStyle = filenode.IconsNerdSimple
	case filenode.IconsNerdSimple:
		m.iconStyle = filenode.IconsNerdFiletype
	case filenode.IconsNerdFiletype:
		m.iconStyle = filenode.IconsNerdFull
	default:
		m.iconStyle = filenode.IconsASCII
	}
	m.fileTree.SetIconStyle(m.iconStyle)
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
				dfCmd := m.diffViewer.SetSize(m.width-m.sidebarWidth(), m.mainContentHeight())
				cmds = append(cmds, dfCmd)
			case "ctrl+c":
				return m, []tea.Cmd{tea.Quit}
			case "enter":
				m.stopSearch()
				dfCmd := m.diffViewer.SetSize(m.width-m.sidebarWidth(), m.mainContentHeight())
				cmds = append(cmds, dfCmd)

				if selected, ok := m.selectedSearchResult(); ok {
					for _, f := range m.files {
						if filenode.GetFileName(f) == selected {
							m.diffViewer, cmd = m.diffViewer.SetFilePatch(f)
							m.fileTree.SetCursorByPath(filenode.GetFileName(f))
							cmds = append(cmds, cmd)
							break
						}
					}
				}

			case "ctrl+n", "down":
				if len(m.filtered) > 0 {
					m.resultsCursor = min(len(m.filtered)-1, m.resultsCursor+1)
					m.resultsVp.ScrollDown(1)
				}
			case "ctrl+p", "up":
				if len(m.filtered) > 0 {
					m.resultsCursor = max(0, m.resultsCursor-1)
					m.resultsVp.ScrollUp(1)
				}
			default:
				m.resultsCursor = 0
			}
		}
		s, sc := m.search.Update(msg)
		cmds = append(cmds, sc)
		m.search = s
		m.setSearchResults()
		m.resultsVp.SetContent(m.resultsView())
	}

	return m, cmds
}

func (m mainModel) View() tea.View {
	var view tea.View
	view.AltScreen = true
	view.MouseMode = tea.MouseModeAllMotion

	view.KeyboardEnhancements.ReportEventTypes = true
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
		if m.isSidebarVisible() {
			sidebarW := m.sidebarWidth()
			rightW := max(m.width-sidebarW, 0)
			leftLine := lipgloss.NewStyle().
				Foreground(leftColor).
				Render(strings.Repeat("─", sidebarW))
			junction := lipgloss.NewStyle().Foreground(leftColor).Render("┬")
			rightLine := lipgloss.NewStyle().
				Foreground(rightColor).
				Render(strings.Repeat("─", rightW))
			separator = leftLine + junction + rightLine
		} else {
			separator = lipgloss.NewStyle().Foreground(rightColor).Render(strings.Repeat("─", m.width))
		}
	}

	sidebar := ""
	if m.isSidebarVisible() {
		searchBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Width(m.sidebarWidth()).
			Render(m.search.View())
		searchBox = zone.Mark(zoneSearchBox, searchBox)

		content := ""
		if m.searching {
			content = zone.Mark(zoneSearchResults, m.resultsVp.View())
		} else {
			content = zone.Mark(zoneFileTree, m.fileTree.View())
		}
		content = lipgloss.NewStyle().
			Render(lipgloss.JoinVertical(lipgloss.Left, searchBox, content))

		sidebar = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(leftColor).Render(content)
	} else {
		// Show a thin grab line when sidebar is hidden.
		// Width(0) means only the border is rendered (1 char).
		grabLine := lipgloss.NewStyle().
			Width(0).
			Height(m.mainContentHeight()-1).
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(lipgloss.Color("8")).
			Render("")
		sidebar = grabLine
	}

	dv := zone.Mark(zoneDiffViewer, m.diffViewer.View())
	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, dv)

	var sections []string

	if !m.config.UI.HideHeader {
		sections = append(sections, m.viewHeader())
	}

	sections = append(sections, separator)
	sections = append(sections, mainContent)

	if !m.config.UI.HideFooter {
		sections = append(sections, m.footerView())
	}

	appView := zone.Scan(lipgloss.JoinVertical(lipgloss.Left, sections...))
	layers := []*lipgloss.Layer{
		lipgloss.NewLayer(appView),
	}

	if m.helpOpen {
		o := m.renderOverlay(m.help.View())
		layers = append(layers, lipgloss.NewLayer(o.rendered).X(o.col).Y(o.row))
	}

	if m.messageOpen {
		o := m.renderOverlay(m.messageViewContent())
		layers = append(layers, lipgloss.NewLayer(o.rendered).X(o.col).Y(o.row))
	}

	comp := lipgloss.NewCompositor(layers...)

	view.Content = comp.Render()

	return view
}

type fileTreeMsg struct {
	files    []*gitdiff.File
	preamble string
	branch   string
}

func (m mainModel) fetchFileTree() tea.Msg {
	// TODO: handle error
	files, preamble, err := gitdiff.Parse(strings.NewReader(m.input + "\n"))
	if err != nil {
		return common.ErrMsg{Err: err}
	}
	sortFiles(files)

	branch := resolveBranch(preamble)
	return fileTreeMsg{files: files, preamble: preamble, branch: branch}
}

// resolveBranch finds branches pointing at the preamble commit.
func resolveBranch(preamble string) string {
	// Check for decoration in commit line: "commit abc123 (HEAD -> branch)"
	for line := range strings.SplitSeq(preamble, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "commit ") {
			if idx := strings.Index(trimmed, " ("); idx > 0 {
				refs := trimmed[idx+2:]
				if end := strings.Index(refs, ")"); end > 0 {
					refs = refs[:end]
				}
				for ref := range strings.SplitSeq(refs, ",") {
					ref = strings.TrimSpace(ref)
					if strings.HasPrefix(ref, "HEAD -> ") {
						return strings.TrimPrefix(ref, "HEAD -> ")
					}
				}
			}
			// Extract hash for --points-at lookup.
			hash := strings.TrimPrefix(trimmed, "commit ")
			if idx := strings.Index(hash, " "); idx > 0 {
				hash = hash[:idx]
			}
			if hash == "" {
				return ""
			}
			out, err := exec.Command("git", "branch", "--points-at", hash).Output()
			if err != nil {
				return ""
			}
			for l := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
				b := strings.TrimLeft(l, " *+")
				if b != "" {
					return b
				}
			}
			return ""
		}
	}
	return ""
}

type commitMeta struct {
	hash   string
	date   string
	author string
}

func (m mainModel) parseCommitMeta() commitMeta {
	var meta commitMeta
	if m.preamble == "" {
		return meta
	}
	for line := range strings.SplitSeq(m.preamble, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "commit ") && meta.hash == "" {
			h := strings.TrimPrefix(trimmed, "commit ")
			// Strip refs decoration if present: "abc123 (HEAD -> branch)"
			if idx := strings.Index(h, " ("); idx > 0 {
				h = h[:idx]
			}
			if len(h) > 7 {
				h = h[:7]
			}
			meta.hash = h
		}
		if strings.HasPrefix(trimmed, "Author:") && meta.author == "" {
			a := strings.TrimPrefix(trimmed, "Author:")
			a = strings.TrimSpace(a)
			if idx := strings.Index(a, " <"); idx > 0 {
				a = a[:idx]
			} else if strings.HasPrefix(a, "<") && strings.Contains(a, "@") {
				// No name, only email: extract username from <user@example.com>
				a = strings.TrimPrefix(a, "<")
				if idx := strings.Index(a, "@"); idx > 0 {
					a = a[:idx]
				}
			}
			parts := strings.Fields(a)
			if len(parts) >= 2 {
				meta.author = string([]rune(parts[0])[:1]) + parts[len(parts)-1]
			} else {
				meta.author = a
			}
		}
		if meta.date == "" {
			for _, prefix := range []string{"AuthorDate:", "Date:"} {
				if strings.HasPrefix(trimmed, prefix) {
					raw := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
					if t, err := time.Parse("Mon Jan 2 15:04:05 2006 -0700", raw); err == nil {
						meta.date = relativeTime(t)
					}
					break
				}
			}
		}
	}
	return meta
}

func (m mainModel) commitSubject() string {
	if m.preamble == "" {
		return ""
	}
	for line := range strings.SplitSeq(m.preamble, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Skip metadata lines (commit hash, Author:, Date:, Merge:, etc.)
		if strings.HasPrefix(trimmed, "commit ") ||
			strings.HasPrefix(trimmed, "Author:") ||
			strings.HasPrefix(trimmed, "AuthorDate:") ||
			strings.HasPrefix(trimmed, "Date:") ||
			strings.HasPrefix(trimmed, "Commit:") ||
			strings.HasPrefix(trimmed, "CommitDate:") ||
			strings.HasPrefix(trimmed, "Merge:") {
			continue
		}
		return trimmed
	}
	return ""
}

func (m mainModel) viewHeader() string {
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).
		Bold(true).
		Render("DIFFNAV")

	sep := lipgloss.NewStyle().Foreground(lipgloss.BrightBlack).Render(" • ")
	hashStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("132"))
	dateStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("172"))
	authorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("109"))
	refStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("072"))

	headerParts := title
	meta := m.cachedMeta
	if meta.hash != "" {
		// Build info segment: hash date author
		var infoParts []string
		infoParts = append(infoParts, hashStyle.Render(meta.hash))
		if meta.date != "" {
			if m.iconStyle != filenode.IconsASCII && m.iconStyle != filenode.IconsUnicode {
				infoParts = append(infoParts, dateStyle.Render(string(md.ClockOutline) + " " + meta.date))
			} else {
				infoParts = append(infoParts, dateStyle.Render(meta.date))
			}
		}
		if meta.author != "" {
			if m.iconStyle != filenode.IconsASCII && m.iconStyle != filenode.IconsUnicode {
				infoParts = append(infoParts, authorStyle.Render(string(md.AccountCircleOutline) + " " + meta.author))
			} else {
			infoParts = append(infoParts, authorStyle.Render(meta.author))
			}
		}
		headerParts = headerParts + sep + strings.Join(infoParts, sep)

		// Branch ref.
		if m.commitBranch != "" {
			branchLabel := "[" + m.commitBranch + "]"
			if m.iconStyle != filenode.IconsASCII && m.iconStyle != filenode.IconsUnicode {
				branchLabel = string(md.SourceBranch) + " " + m.commitBranch
			}
			headerParts = headerParts + sep + refStyle.Render(branchLabel)
		}

		// Commit subject.
		subject := m.commitSubject()
		if subject != "" {
			maxSubjectWidth := m.width - lipgloss.Width(headerParts) - 2
			if maxSubjectWidth > 0 {
				subject = utils.TruncateString(subject, maxSubjectWidth)
				headerParts = headerParts + " " + subject
			}
		}
	}

	return lipgloss.NewStyle().Width(m.width).Render(
		zone.Mark(zoneHeader, headerParts),
	)
}

func (m mainModel) footerView() string {
	base := lipgloss.NewStyle().Background(common.Colors[common.DarkerSelected])
	files := fmt.Sprintf(" %d files", len(m.files))
	sep := lipgloss.NewStyle().Foreground(lipgloss.BrightBlack).Render(" • ")
	added, deleted := m.diffViewer.RootDiffStats()
	help := zone.Mark(zoneHelp, base.Background(lipgloss.BrightBlack).PaddingLeft(1).PaddingRight(1).Render("F1/? help"))
	stats := filenode.ViewDiffStats(added, deleted, base)

	left := lipgloss.JoinHorizontal(lipgloss.Top, files, sep, stats)
	rightWidth := lipgloss.Width(help)

	spacing := base.Render(strings.Repeat(" ", max(0, m.width-lipgloss.Width(left)-rightWidth)))
	return base.
		Width(m.width).
		Height(1).
		Render(lipgloss.JoinHorizontal(lipgloss.Top, left, spacing, help))
}

func (m *mainModel) messageView() string {
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	yellow := lipgloss.NewStyle().Foreground(lipgloss.Yellow)

	var out []string

	// Render preamble lines.
	for line := range strings.SplitSeq(m.preamble, "\n") {
		switch {
		case strings.HasPrefix(line, "commit "):
			out = append(out, dim.Render("commit ")+yellow.Render(strings.TrimPrefix(line, "commit ")))
		case strings.HasPrefix(line, "Author:"),
			strings.HasPrefix(line, "AuthorDate:"),
			strings.HasPrefix(line, "Date:"),
			strings.HasPrefix(line, "Commit:"),
			strings.HasPrefix(line, "CommitDate:"),
			strings.HasPrefix(line, "Merge:"):
			out = append(out, dim.Render(line))
		default:
			out = append(out, line)
		}
	}

	return strings.Join(out, "\n")
}

func (m *mainModel) updateMessageVp() {
	s := overlayStyle()
	maxWidth := min(m.width*3/4, 80)
	maxHeight := max(m.height/2 - s.GetVerticalFrameSize(), 5)
	content := lipgloss.NewStyle().Width(maxWidth).Render(m.messageView())
	m.messageVp.SetWidth(maxWidth)
	m.messageVp.SetHeight(maxHeight)
	m.messageVp.SetContent(content)
}

func (m mainModel) renderScrollbar() string {
	trackHeight := m.messageVp.Height()
	totalLines := m.messageVp.TotalLineCount()
	viewHeight := m.messageVp.Height()

	// Thumb size proportional to visible portion.
	thumbSize := max(1, trackHeight*viewHeight/totalLines)
	// Thumb position based on scroll offset.
	scrollableLines := totalLines - viewHeight
	thumbPos := 0
	if scrollableLines > 0 {
		thumbPos = m.messageVp.YOffset() * (trackHeight - thumbSize) / scrollableLines
		if m.messageVp.YOffset() > 0 && thumbPos == 0 {
			thumbPos = 1
		}
	}

	track := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	thumb := lipgloss.NewStyle().Foreground(lipgloss.Blue)

	var sb strings.Builder
	for i := range trackHeight {
		if i > 0 {
			sb.WriteByte('\n')
		}
		if i >= thumbPos && i < thumbPos+thumbSize {
			sb.WriteString(thumb.Render("┃"))
		} else {
			sb.WriteString(track.Render("│"))
		}
	}
	return sb.String()
}

func (m mainModel) resultsView() string {
	sb := strings.Builder{}
	baseStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F7F7F7"))
	dirStyle := lipgloss.NewStyle().Bold(false).Foreground(lipgloss.Color("#B8B8B8"))
	for i, f := range m.filtered {
		icon := neo.ByPath(f)
		if icon == nil {
			icon = neo.ByFileExtension("txt")
		}
		base := utils.RemoveReset(lipgloss.NewStyle().
			Foreground(icon.Color(true)).
			Render(icon.Glyph().String()) + " " + baseStyle.Render(path.Base(f)))
		dir := utils.TruncateString(
			dirStyle.Render(path.Dir(f)),
			m.config.UI.SearchTreeWidth-2-lipgloss.Width(base),
		)
		if path.Dir(f) == "." {
			dir = ""
		}
		if i == m.resultsCursor {
			bg := lipgloss.NewStyle().Background(lipgloss.Color("#1b1b33"))
			fName := lipgloss.NewStyle().Bold(true).Render(bg.Render(base)) + bg.Render(" ") + bg.Render(dir)
			sb.WriteString(bg.
				Width(m.config.UI.SearchTreeWidth).
				Render(fName) +
				"\n",
			)
		} else {
			fName := base + " " + dir
			sb.WriteString(fName + "\n")
		}
	}
	return sb.String()
}

func (m mainModel) sidebarWidth() int {
	if m.searching {
		return m.config.UI.SearchTreeWidth
	}

	if m.isShowingFileTree {
		return m.fileTree.Width()
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

func (m *mainModel) searchWidth() int {
	return max(0, m.sidebarWidth()-5)
}

func (m *mainModel) stopSearch() {
	m.searching = false
	m.search.SetValue("")
	m.search.Blur()
	m.search.SetWidth(m.searchWidth())
}

func (m mainModel) openInEditor() tea.Cmd {
	if len(m.files) == 0 {
		return nil
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		return nil
	}

	fullpath := m.fileTree.CurrNodePath()
	c := exec.Command(editor, fullpath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return nil
	})
}

// messageViewContent returns the message overlay content (viewport + optional scrollbar).
func (m mainModel) messageViewContent() string {
	vpView := m.messageVp.View()
	if m.messageVp.TotalLineCount() > m.messageVp.Height() {
		vpView = lipgloss.JoinHorizontal(lipgloss.Top, vpView, " ", m.renderScrollbar())
	}
	return vpView
}

// overlayStyle returns the shared style for overlay panels (help, message).
func overlayStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true).
		Padding(1, 3).
		BorderForeground(lipgloss.Blue)
}

type overlayResult struct {
	rendered     string
	col, row     int
	width, height int
}

// renderOverlay renders content with the overlay style and returns the
// rendered string and its position.
func (m mainModel) renderOverlay(content string) overlayResult {
	rendered := overlayStyle().Render(content)
	w := lipgloss.Width(rendered)
	h := lipgloss.Height(rendered)
	return overlayResult{
		rendered: rendered,
		col:      max(0, m.width/2-w/2),
		row:      max(0, m.height/4-2),
		width:    w,
		height:   h,
	}
}

func (m mainModel) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Handle overlays: scroll or click-outside-to-close.
	if m.messageOpen || m.helpOpen {
		switch msg := msg.(type) {
		case tea.MouseClickMsg:
			if msg.Button == tea.MouseLeft {
				var content string
				if m.messageOpen {
					content = m.messageViewContent()
				} else {
					content = m.help.View()
				}
				o := m.renderOverlay(content)
				if msg.X < o.col || msg.X >= o.col+o.width || msg.Y < o.row || msg.Y >= o.row+o.height {
					// Click outside: close overlay.
					m.messageOpen = false
					m.helpOpen = false
					return m, nil
				}
				return m, nil
			}
			return m, nil // Block non-left clicks while overlay is open
		default:
			if m.messageOpen {
				if msg.Mouse().Button == tea.MouseWheelDown {
					m.messageVp.ScrollDown(scrollLines)
				} else if msg.Mouse().Button == tea.MouseWheelUp {
					m.messageVp.ScrollUp(scrollLines)
				}
			}
			return m, nil
		}
	}

	// Handle scroll wheel first.
	if msg.Mouse().Button == tea.MouseWheelUp || msg.Mouse().Button == tea.MouseWheelDown {
		return m.handleScroll(msg)
	}

	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			// Keep coordinate check for resize border (hybrid approach).
			sidebarWidth := m.sidebarWidth()
			if !m.searching && m.isShowingFileTree && abs(msg.X-sidebarWidth) <= sidebarGrabThreshold {
				m.draggingSidebar = true
				return m, nil
			}
			// Allow grabbing the line when sidebar is hidden.
			if !m.isSidebarVisible() && msg.X <= sidebarGrabThreshold {
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
			if zone.Get(zoneHelp).InBounds(msg) {
				m.helpOpen = !m.helpOpen
				m.messageOpen = false
				return m, nil
			}
			if zone.Get(zoneHeader).InBounds(msg) && m.preamble != "" {
				m.messageOpen = !m.messageOpen
				m.helpOpen = false
				if m.messageOpen {
					m.updateMessageVp()
					m.messageVp.GotoTop()
				}
				return m, nil
			}
		}

	case tea.MouseReleaseMsg:
		m.draggingSidebar = false

	case tea.MouseMotionMsg:
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
	clickedIndex := y + m.resultsVp.YOffset()
	if clickedIndex >= len(m.filtered) {
		return m, nil
	}

	// Select the clicked result.
	selected := m.filtered[clickedIndex]
	m.stopSearch()

	var cmd tea.Cmd
	var cmds []tea.Cmd
	dfCmd := m.diffViewer.SetSize(m.width-m.sidebarWidth(), m.mainContentHeight())
	cmds = append(cmds, dfCmd)

	for _, f := range m.files {
		if filenode.GetFileName(f) == selected {
			m.diffViewer, cmd = m.diffViewer.SetFilePatch(f)
			m.fileTree.SetCursorByPath(filenode.GetFileName(f))
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
	m.search.SetWidth(m.searchWidth())
	m.search.SetValue("")
	m.resultsCursor = 0
	m.setSearchResults()

	m.resultsVp.SetWidth(m.config.UI.SearchTreeWidth)
	m.resultsVp.SetHeight(m.mainContentHeight() - searchHeight)
	m.resultsVp.SetContent(m.resultsView())

	dfCmd := m.diffViewer.SetSize(m.width-m.sidebarWidth(), m.mainContentHeight())
	return m, tea.Batch(dfCmd, m.search.Focus())
}

func (m mainModel) handleFileTreeClick(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Use zone-relative coordinates.
	_, y := zone.Get(zoneFileTree).Pos(msg)
	if y < 0 {
		return m, nil
	}
	clickedY := y + m.fileTree.ViewportYOffset()

	node := m.fileTree.GetNodeAtY(clickedY)
	if node == nil {
		return m, nil
	}

	var cmd tea.Cmd
	m, cmd = m.setNodeDiff(node)
	// Use SetCursorNoScroll to avoid jumping the file tree view.
	m.fileTree.SetCursorNoScroll(node.YOffset())
	return m, cmd
}

func (m mainModel) handleScroll(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	lines := scrollLines

	// Check if scrolling in sidebar (file tree or search results).
	if zone.Get(zoneFileTree).InBounds(msg) || zone.Get(zoneSearchResults).InBounds(msg) {
		if msg.Mouse().Button == tea.MouseWheelUp {
			if m.searching {
				m.resultsVp.ScrollUp(lines)
			} else {
				m.fileTree.ScrollUp(lines)
			}
		} else {
			if m.searching {
				m.resultsVp.ScrollDown(lines)
			} else {
				m.fileTree.ScrollDown(lines)
			}
		}
		return m, nil
	}

	// Check if scrolling in diff viewer.
	if zone.Get(zoneDiffViewer).InBounds(msg) {
		if msg.Mouse().Button == tea.MouseWheelUp {
			m.diffViewer.ScrollUp(lines)
		} else {
			m.diffViewer.ScrollDown(lines)
		}
	}
	return m, nil
}

func (m mainModel) handleSidebarDrag(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.searching {
		m.draggingSidebar = false
		return m, nil
	}

	// Hide sidebar if dragged below threshold.
	if msg.Mouse().X < sidebarHideWidth {
		m.isShowingFileTree = false
		m.draggingSidebar = false
		cmd := m.diffViewer.SetSize(m.width, m.mainContentHeight())
		return m, cmd
	}

	// Clamp to reasonable bounds.
	minWidth := sidebarMinWidth
	maxWidth := m.width / 2
	newWidth := max(minWidth, min(maxWidth, msg.Mouse().X))

	// TODO: for some reason setting a value smaller than minResizeStep
	// will garble up the output when resizing. I have no idea why.
	if abs(newWidth-m.sidebarWidth()) < minResizeStep {
		return m, nil
	}

	// Resize components.
	cmds := []tea.Cmd{}

	cmds = append(cmds, m.diffViewer.SetSize(m.width-newWidth, m.mainContentHeight()))
	m.fileTree.SetSize(newWidth-1, m.mainContentHeight()-searchHeight-1)

	return m, tea.Batch(cmds...)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (m mainModel) moveToFile(movement int) (mainModel, tea.Cmd) {
	var cmd tea.Cmd
	var moved bool
	switch movement {
	case -1:
		moved = m.fileTree.PrevFile()
	case 1:
		moved = m.fileTree.NextFile()
	}

	if !moved {
		return m, nil
	}

	node := m.fileTree.GetCurrNode()
	m, cmd = m.setNodeDiff(node)
	m.diffViewer.GoToTop()

	return m, cmd
}

func (m mainModel) moveCursor(movement int) (mainModel, tea.Cmd) {
	var cmd tea.Cmd
	switch movement {
	case -1:
		m.fileTree.Up()
	case 1:
		m.fileTree.Down()
	}

	node := m.fileTree.GetCurrNode()
	m, cmd = m.setNodeDiff(node)
	m.diffViewer.GoToTop()

	return m, cmd
}

func (m mainModel) setNodeDiff(node *tree.Node) (mainModel, tea.Cmd) {
	var cmd tea.Cmd
	switch val := node.GivenValue().(type) {
	case *filenode.FileNode:
		m.diffViewer, cmd = m.diffViewer.SetFilePatch(val.File)
	case string, *dirnode.DirNode:
		files := m.fileTree.GetCurrNodeDesendantDiffs()

		fullPath := "/"
		if val, ok := node.GivenValue().(*dirnode.DirNode); ok {
			fullPath = val.FullPath
		}
		m.diffViewer, cmd = m.diffViewer.SetDirPatch(fullPath, files)
	}

	return m, cmd
}

func (m *mainModel) setSearchResults() {
	filtered := make([]string, 0)
	for _, f := range m.files {
		if strings.Contains(
			strings.ToLower(filenode.GetFileName(f)),
			strings.ToLower(m.search.Value()),
		) {
			filtered = append(filtered, filenode.GetFileName(f))
		}
	}
	m.filtered = filtered
	switch {
	case len(m.filtered) == 0:
		m.resultsCursor = 0
	case m.resultsCursor < 0:
		m.resultsCursor = 0
	case m.resultsCursor >= len(m.filtered):
		m.resultsCursor = len(m.filtered) - 1
	}
}

func (m mainModel) selectedSearchResult() (string, bool) {
	if len(m.filtered) == 0 {
		return "", false
	}
	if m.resultsCursor < 0 || m.resultsCursor >= len(m.filtered) {
		return "", false
	}
	return m.filtered[m.resultsCursor], true
}

func (m mainModel) isSidebarVisible() bool {
	return m.isShowingFileTree || m.searching
}
