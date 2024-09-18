package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

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
		m.vp.Height = m.height
		log.Printf("width: %d, height: %d", m.width, m.height)
		cmds = append(cmds, diff(m.file, m.width))
	}

	return m, tea.Batch(cmds...)
}

func (m diffModel) View() string {
	if m.buffer == nil {
		return "Loading..."
	}
	return m.vp.View()
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
		deltac := exec.Command("delta", "--side-by-side", "--paging=never", fmt.Sprintf("-w=%d", width))
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
