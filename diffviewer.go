package main

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
)

const headerHeight = 3

type diffModel struct {
	vp     viewport.Model
	buffer *bytes.Buffer
	width  int
	height int
	file   *gitdiff.File
}

func initialDiffModel() diffModel {
	return diffModel{
		vp: viewport.Model{},
	}
}

func (m diffModel) Init() tea.Cmd {
	return nil
}

func (m diffModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		m.vp.SetContent(msg.text)
	case dimensionsMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.vp.Width = m.width
		m.vp.Height = m.height - headerHeight
		cmds = append(cmds, diff(m.file, m.width))
	}

	return m, tea.Batch(cmds...)
}

func (m diffModel) View() string {
	if m.buffer == nil {
		return "Loading..."
	}
	return lipgloss.JoinVertical(lipgloss.Left, m.headerView(), m.vp.View())
}

func (m diffModel) headerView() string {
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
		Width(m.width).
		PaddingLeft(1).
		Height(headerHeight - 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("8")).
		Render(lipgloss.JoinVertical(lipgloss.Left, top, bottom))
}

func (m diffModel) SetFilePatch(file *gitdiff.File) (diffModel, tea.Cmd) {
	m.buffer = new(bytes.Buffer)
	m.file = file
	return m, diff(m.file, m.width)
}

func diff(file *gitdiff.File, width int) tea.Cmd {
	if width == 0 || file == nil {
		return nil
	}
	return func() tea.Msg {
		sideBySide := !file.IsNew && !file.IsDelete
		args := []string{"--paging=never", fmt.Sprintf("-w=%d", width)}
		if sideBySide {
			args = append(args, "--side-by-side")
		}
		deltac := exec.Command("delta", args...)
		deltac.Env = os.Environ()
		deltac.Stdin = strings.NewReader(file.String() + "\n")
		out, err := deltac.Output()

		if err != nil {
			return errMsg{err}
		}

		return diffContentMsg{text: string(out)}
	}
}

type diffContentMsg struct {
	text string
}
