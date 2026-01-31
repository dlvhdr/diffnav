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

type Model struct {
	common.Common
	vp         viewport.Model
	file       *gitdiff.File
	dir        string
	dirFiles   []*gitdiff.File
	sideBySide bool
}

func New(sideBySide bool) Model {
	return Model{
		vp:         viewport.Model{},
		sideBySide: sideBySide,
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
		m.vp.SetContent(strings.Join(lines, "\n"))
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
	return m.diff()
}

func (m *Model) diff() tea.Cmd {
	if m.file != nil {
		return diffFile(m.file, m.Width, m.sideBySide)
	} else if m.dir != "" {
		return diffDir(m.dir, m.dirFiles, m.Width, m.sideBySide)
	}

	return nil
}

func (m Model) headerView() string {
	if m.dir != "" {
		return m.dirHeaderView()
	}

	if m.file == nil {
		return ""
	}
	name := filenode.GetFileName(m.file)
	base := lipgloss.NewStyle()

	fileIcon := icons.GetIcon(name, false)
	prefix := base.Render(fileIcon) + base.Render(" ")
	name = utils.TruncateString(name, m.Width-lipgloss.Width(prefix))
	top := prefix + base.Bold(true).Render(name)

	bottom := filenode.ViewFileLinesCounts(m.file, base)

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
	name := utils.TruncateString(m.dir, m.Width-lipgloss.Width(prefix))

	var additions, deletions int64
	for _, file := range m.dirFiles {
		a, d := filenode.LinesCounts(file)
		additions += a
		deletions += d
	}

	top := prefix + base.Bold(true).Render(name)
	bottom := filenode.ViewLinesCounts(additions, deletions, base)
	return base.
		Width(m.Width).
		Height(dirHeaderHeight - 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("8")).
		Render(lipgloss.JoinVertical(lipgloss.Left, top, bottom))
}

func (m Model) SetFilePatch(file *gitdiff.File) (Model, tea.Cmd) {
	m.file = file
	m.dir = ""
	return m, diffFile(m.file, m.Width, m.sideBySide)
}

func (m Model) SetDirPatch(dirPath string, files []*gitdiff.File) (Model, tea.Cmd) {
	m.file = nil
	m.dir = dirPath
	m.dirFiles = files
	return m, diffDir(dirPath, files, m.Width, m.sideBySide)
}

func (m *Model) GoToTop() {
	m.vp.GotoTop()
}

// SetSideBySide updates the diff view mode and re-renders.
func (m *Model) SetSideBySide(sideBySide bool) tea.Cmd {
	m.sideBySide = sideBySide
	return diffFile(m.file, m.Width, m.sideBySide)
}

// ScrollUp scrolls the viewport up by the given number of lines.
func (m *Model) ScrollUp(lines int) {
	m.vp.ScrollUp(lines)
}

// ScrollDown scrolls the viewport down by the given number of lines.
func (m *Model) ScrollDown(lines int) {
	m.vp.ScrollDown(lines)
}

func diffFile(file *gitdiff.File, width int, sideBySidePreference bool) tea.Cmd {
	if width == 0 || file == nil {
		return nil
	}
	return func() tea.Msg {
		// Only use side-by-side if preference is true AND file is not new/deleted
		useSideBySide := sideBySidePreference && !file.IsNew && !file.IsDelete
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

		return diffContentMsg{text: string(out)}
	}
}

func diffDir(dirPath string, files []*gitdiff.File, width int, sideBySidePreference bool) tea.Cmd {
	if width == 0 || dirPath == "" {
		return nil
	}
	return func() tea.Msg {
		// Only use side-by-side if preference is true AND file is not new/deleted
		s := common.BgStyles[common.Selected]
		c := common.LipglossColorToHex(common.Colors[common.Selected])
		useSideBySide := sideBySidePreference
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
		for _, file := range files {
			strs.WriteString(file.String())
		}
		deltac.Stdin = strings.NewReader(strs.String() + "\n")
		out, err := deltac.Output()
		if err != nil {
			return common.ErrMsg{Err: err}
		}

		return diffContentMsg{text: string(out)}
	}
}

type diffContentMsg struct {
	text string
}
