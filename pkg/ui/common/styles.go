package common

import (
	"fmt"
	"image/color"

	"charm.land/lipgloss/v2"
)

type Key int

// Available colors.
const (
	Selected Key = iota
	DarkerSelected
)

var Colors = map[Key]color.RGBA{
	Selected:       {R: 0x2d, G: 0x2c, B: 0x35, A: 0xFF}, // "#2d2c35"
	DarkerSelected: {R: 0x20, G: 0x1F, B: 0x26, A: 0xFF}, // "#201F26"
}

var BgStyles = map[Key]lipgloss.Style{
	Selected:       lipgloss.NewStyle().Background(Colors[Selected]),
	DarkerSelected: lipgloss.NewStyle().Background(Colors[DarkerSelected]),
}

// lipglossColorToHex converts a color.Color to hex string
func LipglossColorToHex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}
