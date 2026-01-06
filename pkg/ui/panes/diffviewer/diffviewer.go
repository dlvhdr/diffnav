package diffviewer

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/dlvhdr/diffnav/pkg/ui/common"
)

const dirHeaderHeight = 3

type Model struct {
	common.Common
	vp     viewport.Model
	buffer *bytes.Buffer
	file   *gitdiff.File
}

func New() Model {
	return Model{
		vp: viewport.Model{},
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
			if lipgloss.Width(line) > m.vp.Width && m.vp.Width > 0 {
				lines[i] = ansi.Truncate(line, m.vp.Width, "")
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
	m.vp.Width = m.Width
	m.vp.Height = m.Height - dirHeaderHeight
	return diff(m.file, m.Width)
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

	var added int64 = 0
	var deleted int64 = 0
	frags := m.file.TextFragments
	for _, frag := range frags {
		added += frag.LinesAdded
		deleted += frag.LinesDeleted
	}

	top := lipgloss.JoinHorizontal(lipgloss.Top, base.Render(""), base.Render(" "), base.Bold(true).Render(name))
	bottom := lipgloss.JoinHorizontal(
		lipgloss.Top,
		base.Foreground(lipgloss.Color("2")).Render(fmt.Sprintf("  +%d ", added)),
		base.Foreground(lipgloss.Color("1")).Render(fmt.Sprintf("-%d", deleted)),
	)

	return base.
		Width(m.Width).
		PaddingLeft(1).
		Height(dirHeaderHeight - 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("8")).
		Render(lipgloss.JoinVertical(lipgloss.Left, top, bottom))
}

func (m Model) SetFilePatch(file *gitdiff.File) (Model, tea.Cmd) {
	m.buffer = new(bytes.Buffer)
	m.file = file
	return m, diff(m.file, m.Width)
}

func (m *Model) GoToTop() {
	m.vp.GotoTop()
}

func diff(file *gitdiff.File, width int) tea.Cmd {
	if width == 0 || file == nil {
		return nil
	}
	return func() tea.Msg {
		sideBySide := !file.IsNew && !file.IsDelete
		args := []string{
			"--paging=never",
			fmt.Sprintf("-w=%d", width),
			fmt.Sprintf("--max-line-length=%d", width),
		}
		if sideBySide {
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
