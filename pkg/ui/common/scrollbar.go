package common

import (
	"math"
	"strings"

	"charm.land/lipgloss/v2"
)

type ScrollbarStyles struct {
	Thumb lipgloss.Style
	Track lipgloss.Style
}

type Scrollbar struct {
	Styles ScrollbarStyles
}

// RenderScrollbar renders a vertical scrollbar track with a thumb indicator.
// It takes the viewport height, total number of items, and the current
// scroll item idx. Returns an empty string if all content fits.
func (m Scrollbar) View(trackHeight, totalItems, firstItemIdx int, lastItemIdx int) string {
	if totalItems <= trackHeight {
		return ""
	}

	scroll := float64(1) / float64(totalItems-trackHeight)
	thumbSize := max(1, int(math.Floor(float64(trackHeight)*scroll)))
	if thumbSize > 1 && thumbSize+2 > trackHeight {
		thumbSize -= 1
	}

	scrollableLines := totalItems - trackHeight
	thumbPos := 0
	if scrollableLines > 0 {
		thumbPos = int(math.Min(
			float64(trackHeight-thumbSize),
			math.Floor(
				float64(firstItemIdx)*(float64(trackHeight-thumbSize))/float64(scrollableLines),
			),
		))
		if firstItemIdx == 0 {
			thumbPos = 0
		} else if firstItemIdx > 0 && thumbPos == 0 {
			// put thumb almost at the top-most position
			thumbPos = 1
		} else if lastItemIdx < totalItems-1 && thumbPos+thumbSize == trackHeight {
			// put thumb almost at the bottom-most position
			thumbPos = thumbPos - 1
		} else if firstItemIdx+trackHeight > totalItems-1 && thumbPos+thumbSize < trackHeight {
			// put thumb at the bottom-most position
			thumbPos = trackHeight - thumbSize
		}
	}

	var sb strings.Builder
	for i := range trackHeight {
		if i > 0 {
			sb.WriteByte('\n')
		}
		if i >= thumbPos && i < thumbPos+thumbSize {
			sb.WriteString(m.Styles.Thumb.Render("┃"))
		} else {
			sb.WriteString(m.Styles.Track.Render("│"))
		}
	}
	return sb.String()
}
