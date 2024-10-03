package common

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Common is a struct that contains the width and height of a component.
type Common struct {
	Width, Height int
}

type Component interface {
	SetSize(width, height int) tea.Cmd
}
