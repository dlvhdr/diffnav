package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

type diffModel struct {
	buffer *bytes.Buffer
	width  int
	height int
	path   string
	text   string
}

func initialDiffModel() diffModel {
	return diffModel{
		text: "",
	}
}

func (m diffModel) Init() tea.Cmd {
	return nil
}

func (m diffModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case diffContentMsg:
		m.text = msg.text
	case tea.WindowSizeMsg:
		m.width = msg.Width - fileTreeWidth
		m.height = msg.Height

	}

	return m, nil
}

func (m diffModel) View() string {
	if m.buffer == nil {
		return "Loading..."
	}
	return m.text
}

func (m diffModel) SetFilePath(path string) (diffModel, tea.Cmd) {
	m.buffer = new(bytes.Buffer)
	m.path = path
	return m, diff(m.path, m.width)
}

func diff(path string, width int) tea.Cmd {
	return func() tea.Msg {
		var outb bytes.Buffer
		gitc := exec.Command("git", "diff", path)
		deltac := exec.Command("delta", "--side-by-side", "--paging=never", fmt.Sprintf("-w=%d", width))
		deltac.Env = os.Environ()
		deltac.Stdin, _ = gitc.StdoutPipe()
		deltac.Stdout = &outb
		_ = deltac.Start()
		_ = gitc.Run()
		_ = deltac.Wait()
		return diffContentMsg{text: string(outb.String())}
	}
}

type diffContentMsg struct {
	text string
}
