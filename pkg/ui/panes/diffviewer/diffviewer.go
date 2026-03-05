package diffviewer

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/charmbracelet/x/ansi"

	"github.com/dlvhdr/diffnav/pkg/filenode"
	"github.com/dlvhdr/diffnav/pkg/icons"
	"github.com/dlvhdr/diffnav/pkg/ui/common"
	"github.com/dlvhdr/diffnav/pkg/utils"
)

const dirHeaderHeight = 3

type cachedNode struct {
	path      string
	files     []*gitdiff.File
	additions int64
	deletions int64
	diff      string
}

type nodeCache map[string]*cachedNode

func cacheKey(path string, sideBySide bool) string {
	if sideBySide {
		return path + ":sbs"
	}
	return path
}

type Model struct {
	common.Common
	vp         viewport.Model
	file       *cachedNode
	dir        *cachedNode
	cache      nodeCache
	sideBySide bool
	preamble   string
}

// SetPreamble stores the preamble text (e.g. commit metadata from git show).
func (m *Model) SetPreamble(preamble string) {
	m.preamble = preamble
}

func New(sideBySide bool) Model {
	return Model{
		vp:         viewport.Model{},
		sideBySide: sideBySide,
		cache:      map[string]*cachedNode{},
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "down", "j":
			break
		case "up", "k":
			break
		default:
			vp, vpCmd := m.vp.Update(msg)
			cmds = append(cmds, vpCmd)
			m.vp = vp
		}

	case diffContentMsg:
		// Truncate lines to viewport width to prevent ANSI escape overflow.
		lines := strings.Split(msg.text, "\n")
		for i, line := range lines {
			if lipgloss.Width(line) > m.vp.Width() && m.vp.Width() > 0 {
				lines[i] = ansi.Truncate(line, m.vp.Width(), "")
			}
		}
		diff := strings.Join(lines, "\n")
		if _, ok := m.cache[msg.cacheKey]; ok {
			m.cache[msg.cacheKey].diff = diff
		}
		m.vp.SetContent(diff)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return lipgloss.JoinVertical(lipgloss.Left, m.headerView(), m.vp.View())
}

func (m *Model) SetSize(width, height int) tea.Cmd {
	m.Width = width
	m.Height = height
	m.vp.SetWidth(m.Width)
	m.vp.SetHeight(m.Height - dirHeaderHeight)
	m.cache = make(nodeCache)
	return m.diff()
}

func (m *Model) diff() tea.Cmd {
	if m.file != nil {
		key := cacheKey(m.file.path, m.sideBySide)
		if cached, ok := m.cache[key]; ok && cached.diff != "" {
			m.file = cached
			m.vp.SetContent(cached.diff)
			return nil
		}
		node := &cachedNode{
			path:      m.file.path,
			files:     m.file.files,
			additions: m.file.additions,
			deletions: m.file.deletions,
		}
		m.file = node
		m.cache[key] = node
		return diffFile(node, m.Width, m.sideBySide)
	} else if m.dir != nil {
		key := cacheKey(m.dir.path, m.sideBySide)
		if cached, ok := m.cache[key]; ok && cached.diff != "" {
			m.dir = cached
			m.vp.SetContent(cached.diff)
			return nil
		}
		node := &cachedNode{
			path:      m.dir.path,
			files:     m.dir.files,
			additions: m.dir.additions,
			deletions: m.dir.deletions,
		}
		m.dir = node
		m.cache[key] = node
		preamble := ""
		if m.dir.path == "/" {
			preamble = m.preamble
		}
		return diffDir(node, m.Width, m.sideBySide, preamble)
	}

	return nil
}

func (m Model) headerView() string {
	if m.dir != nil {
		return m.dirHeaderView()
	}

	if m.file == nil || len(m.file.files) != 1 {
		return ""
	}
	name := m.file.path
	base := lipgloss.NewStyle()

	fileIcon := icons.GetIcon(name, false)
	prefix := base.Render(fileIcon) + base.Render(" ")
	name = utils.TruncateString(name, m.Width-lipgloss.Width(prefix))
	top := prefix + base.Bold(true).Render(name)

	bottom := filenode.ViewFileDiffStats(m.file.files[0], base)

	return base.
		Width(m.Width).
		Height(dirHeaderHeight - 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("8")).
		Render(lipgloss.JoinVertical(lipgloss.Left, top, bottom))
}

func (m Model) dirHeaderView() string {
	base := lipgloss.NewStyle().Foreground(lipgloss.Blue)
	prefix := base.Render(" ")
	name := utils.TruncateString(m.dir.path, m.Width-lipgloss.Width(prefix))

	top := prefix + base.Bold(true).Render(name)
	bottom := filenode.ViewDiffStats(m.dir.additions, m.dir.deletions, base)
	return base.
		Width(m.Width).
		Height(dirHeaderHeight - 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("8")).
		Render(lipgloss.JoinVertical(lipgloss.Left, top, bottom))
}

func (m Model) SetFilePatch(file *gitdiff.File) (Model, tea.Cmd) {
	m.dir = nil

	fname := filenode.GetFileName(file)
	key := cacheKey(fname, m.sideBySide)
	if cached, ok := m.cache[key]; ok {
		m.file = cached
		m.vp.SetContent(cached.diff)
		return m, nil
	}

	files := make([]*gitdiff.File, 1)
	files[0] = file
	additions, deletions := filenode.DiffStats(file)
	m.file = &cachedNode{
		path:      fname,
		files:     files,
		additions: additions,
		deletions: deletions,
	}
	m.cache[key] = m.file

	return m, diffFile(m.file, m.Width, m.sideBySide)
}

func (m Model) SetDirPatch(dirPath string, files []*gitdiff.File) (Model, tea.Cmd) {
	m.file = nil

	key := cacheKey(dirPath, m.sideBySide)
	if cached, ok := m.cache[key]; ok {
		m.dir = cached
		m.vp.SetContent(cached.diff)
		return m, nil
	}

	var added, deleted int64
	for _, file := range files {
		na, nd := filenode.DiffStats(file)
		added += na
		deleted += nd
	}
	m.dir = &cachedNode{
		path:      dirPath,
		files:     files,
		additions: added,
		deletions: deleted,
	}
	m.cache[key] = m.dir
	preamble := ""
	if dirPath == "/" {
		preamble = m.preamble
	}
	return m, diffDir(m.dir, m.Width, m.sideBySide, preamble)
}

func (m *Model) GoToTop() {
	m.vp.GotoTop()
}

// SetSideBySide updates the diff view mode and re-renders.
func (m *Model) SetSideBySide(sideBySide bool) tea.Cmd {
	m.sideBySide = sideBySide
	return m.diff()
}

// ScrollUp scrolls the viewport up by the given number of lines.
func (m *Model) ScrollUp(lines int) {
	m.vp.ScrollUp(lines)
}

// ScrollDown scrolls the viewport down by the given number of lines.
func (m *Model) ScrollDown(lines int) {
	m.vp.ScrollDown(lines)
}

func diffFile(node *cachedNode, width int, sideBySide bool) tea.Cmd {
	if width == 0 || node == nil || len(node.files) != 1 {
		return nil
	}

	file := node.files[0]
	key := cacheKey(node.path, sideBySide)
	return func() tea.Msg {
		// Only use side-by-side if preference is true AND file is not new/deleted
		useSideBySide := sideBySide && !file.IsNew && !file.IsDelete
		args := []string{
			"--paging=never",
			fmt.Sprintf("-w=%d", width),
			fmt.Sprintf("--max-line-length=%d", width),
		}
		if useSideBySide {
			args = append(args, "--side-by-side")
		}
		deltac := exec.Command("delta", args...)
		deltac.Env = os.Environ()
		deltac.Stdin = strings.NewReader(file.String() + "\n")
		out, err := deltac.Output()
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return diffContentMsg{cacheKey: key, text: string(out)}
	}
}

func diffDir(dir *cachedNode, width int, sideBySide bool, preamble string) tea.Cmd {
	if width == 0 || dir == nil {
		return nil
	}
	key := cacheKey(dir.path, sideBySide)
	return func() tea.Msg {
		s := common.BgStyles[common.Selected]
		c := common.LipglossColorToHex(common.Colors[common.Selected])
		useSideBySide := sideBySide
		args := []string{
			"--paging=never",
			fmt.Sprintf("--file-modified-label=%s",
				utils.RemoveReset(s.Foreground(lipgloss.Yellow).Render(" "))),
			fmt.Sprintf("--file-removed-label=%s",
				utils.RemoveReset(s.Foreground(lipgloss.Red).Render(" "))),
			fmt.Sprintf("--file-added-label=%s",
				utils.RemoveReset(s.Foreground(lipgloss.Green).Render(" "))),
			fmt.Sprintf("--file-style='%s bold %s'", c, c),
			fmt.Sprintf("--file-decoration-style='%s box %s'", c, c),
			fmt.Sprintf("-w=%d", width),
			fmt.Sprintf("--max-line-length=%d", width),
		}
		if useSideBySide {
			args = append(args, "--side-by-side")
		}
		deltac := exec.Command("delta", args...)
		deltac.Env = os.Environ()
		strs := strings.Builder{}
		for _, file := range dir.files {
			strs.WriteString(file.String())
		}
		deltac.Stdin = strings.NewReader(strs.String() + "\n")
		out, err := deltac.Output()
		if err != nil {
			return common.ErrMsg{Err: err}
		}

		text := string(out)
		if preamble != "" {
			text = renderPreamble(preamble) + "\n" + text
		}
		return diffContentMsg{cacheKey: key, text: text}
	}
}

func renderPreamble(preamble string) string {
	preamble = strings.TrimSpace(preamble)
	if preamble == "" {
		return ""
	}

	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	yellow := lipgloss.NewStyle().Foreground(lipgloss.Yellow)

	var out []string
	for _, line := range strings.Split(preamble, "\n") {
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

type diffContentMsg struct {
	cacheKey string
	text     string
}

func (m *Model) RootDiffStats() (int64, int64) {
	if item, ok := m.cache[cacheKey("/", m.sideBySide)]; ok {
		return item.additions, item.deletions
	}

	return 0, 0
}
