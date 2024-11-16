package common

import (
	tea "charm.land/bubbletea/v2"
)

// Common is a struct that contains the width and height of a component.
type Common struct {
	Width, Height int
}

type Component interface {
	SetSize(width, height int) tea.Cmd
}
