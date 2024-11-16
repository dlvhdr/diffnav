package diffviewer

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/charmbracelet/x/ansi"

	"github.com/dlvhdr/diffnav/pkg/ui/common"
	"github.com/dlvhdr/diffnav/pkg/utils"
)

const dirHeaderHeight = 3

type Model struct {
	common.Common
	vp         viewport.Model
	buffer     *bytes.Buffer
	file       *gitdiff.File
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
	if m.buffer == nil {
		return "Loading..."
	}
	return lipgloss.JoinVertical(lipgloss.Left, m.headerView(), m.vp.View())
}

func (m *Model) SetSize(width, height int) tea.Cmd {
	m.Width = width
	m.Height = height
	m.vp.SetWidth(m.Width)
	m.vp.SetHeight(m.Height - dirHeaderHeight)
	return diff(m.file, m.Width, m.sideBySide)
}

func (m Model) headerView() string {
	if m.file == nil {
		return ""
	}
	name := m.file.NewName
	if name == "" {
		name = m.file.OldName
	}

	base := lipgloss.NewStyle()
	prefix := base.Render("") + base.Render(" ")
	name = utils.TruncateString(name, m.Width-lipgloss.Width(prefix))
	top := prefix + base.Bold(true).Render(name)

	var added int64 = 0
	var deleted int64 = 0
	frags := m.file.TextFragments
	for _, frag := range frags {
		added += frag.LinesAdded
		deleted += frag.LinesDeleted
	}

	bottom := lipgloss.JoinHorizontal(
		lipgloss.Top,
		base.Foreground(lipgloss.Color("2")).Render(fmt.Sprintf("  +%d ", added)),
		base.Foreground(lipgloss.Color("1")).Render(fmt.Sprintf("-%d", deleted)),
	)

	return base.
		Width(m.Width).
		Height(dirHeaderHeight - 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("8")).
		Render(lipgloss.JoinVertical(lipgloss.Left, top, bottom))
}

func (m Model) SetFilePatch(file *gitdiff.File) (Model, tea.Cmd) {
	m.buffer = new(bytes.Buffer)
	m.file = file
	return m, diff(m.file, m.Width, m.sideBySide)
}

func (m *Model) GoToTop() {
	m.vp.GotoTop()
}

// SetSideBySide updates the diff view mode and re-renders.
func (m *Model) SetSideBySide(sideBySide bool) tea.Cmd {
	m.sideBySide = sideBySide
	return diff(m.file, m.Width, m.sideBySide)
}

// ScrollUp scrolls the viewport up by the given number of lines.
func (m *Model) ScrollUp(lines int) {
	m.vp.ScrollUp(lines)
}

// ScrollDown scrolls the viewport down by the given number of lines.
func (m *Model) ScrollDown(lines int) {
	m.vp.ScrollDown(lines)
}

func diff(file *gitdiff.File, width int, sideBySidePreference bool) tea.Cmd {
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

type diffContentMsg struct {
	text string
}
