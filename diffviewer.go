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
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case diffContentMsg:
		m.vp.SetContent(msg.text)
	case tea.WindowSizeMsg:
		m.width = msg.Width - fileTreeWidth
		m.height = msg.Height
		m.vp.Width = m.width
		m.vp.Height = m.height
		log.Printf("width: %d, height: %d", m.width, m.height)
		cmd = diff(m.file, m.width)

	case tea.MouseMsg:
		m.vp, cmd = m.vp.Update(msg)
	}

	return m, cmd
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
		deltac := exec.Command("delta", "--side-by-side", "--paging=never", "--config=cfg/delta.conf", fmt.Sprintf("-w=%d", width))
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
