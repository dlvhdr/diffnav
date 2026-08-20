package themepicker

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/dlvhdr/diffnav/pkg/ui/list"
	"github.com/rivo/uniseg"
	"github.com/sahilm/fuzzy"
)

// ListItem represents a selectable and searchable item in a dialog list.
type ListItem interface {
	list.FilterableItem
	list.Focusable
	list.MatchSettable

	// ID returns the unique identifier of the item.
	ID() string
}

// sameFuzzyMatch reports whether two fuzzy.Match values are
// observably equal. Because Match contains a slice (MatchedIndexes)
// it is not directly comparable with ==; we compare the scalar
// fields and then walk the indexes. The dialog list items use this
// to skip gratuitous version bumps when SetMatch reapplies the same
// match.
func sameFuzzyMatch(a, b fuzzy.Match) bool {
	return a.Str == b.Str &&
		a.Index == b.Index &&
		a.Score == b.Score &&
		slices.Equal(a.MatchedIndexes, b.MatchedIndexes)
}

func renderItem(
	iStyles ListItemStyles,
	title string,
	info string,
	focused bool,
	width int,
	cache map[int]string,
	m *fuzzy.Match,
) string {
	if cache == nil {
		cache = make(map[int]string)
	}

	cached, ok := cache[width]
	if ok {
		return cached
	}

	style := iStyles.ItemBlurred
	if focused {
		style = iStyles.ItemFocused
	}

	// Build content to the width left after the item style's own padding,
	// so the final rendered line is exactly `width` wide. Otherwise the
	// padding pushes the line past the list width and it wraps.
	lineWidth := max(0, width-style.GetHorizontalFrameSize())

	var infoText string
	var infoWidth int
	if len(info) > 0 {
		// Cap the info column so a long value (e.g. a provider name) can
		// truncate instead of overflowing the row or squeezing the title
		// to nothing; the title keeps at least half the width.
		if maxInfo := lineWidth / 2; lipgloss.Width(info)+2 > maxInfo {
			info = ansi.Truncate(info, max(0, maxInfo-2), "…")
		}
		infoText = fmt.Sprintf(" %s", info)
		if focused {
			infoText = iStyles.InfoTextFocused.Render(infoText)
		} else {
			infoText = iStyles.InfoTextBlurred.Render(infoText)
		}

		infoWidth = lipgloss.Width(infoText)
	}

	title = ansi.Truncate(title, max(0, lineWidth-infoWidth), "…")
	titleWidth := lipgloss.Width(title)
	gap := strings.Repeat(" ", max(0, lineWidth-titleWidth-infoWidth))
	content := title
	if m != nil && len(m.MatchedIndexes) > 0 {
		var lastPos int
		parts := make([]string, 0)
		ranges := matchedRanges(m.MatchedIndexes)
		for _, rng := range ranges {
			start, stop := bytePosToVisibleCharPos(title, rng)
			if start > lastPos {
				parts = append(parts, ansi.Cut(title, lastPos, start))
			}
			// NOTE: We're using [ansi.Style] here instead of [lipglosStyle]
			// because we can control the underline start and stop more
			// precisely via [ansi.AttrUnderline] and [ansi.AttrNoUnderline]
			// which only affect the underline attribute without interfering
			// with other style attributes.
			parts = append(
				parts,
				ansi.NewStyle().Underline(true).String(),
				ansi.Cut(title, start, stop+1),
				ansi.NewStyle().Underline(false).String(),
			)
			lastPos = stop + 1
		}
		if lastPos < ansi.StringWidth(title) {
			parts = append(parts, ansi.Cut(title, lastPos, ansi.StringWidth(title)))
		}

		content = strings.Join(parts, "")
	}

	content = style.Render(content + gap + infoText)
	cache[width] = content
	return content
}

type ListItemStyles struct {
	ItemBlurred     lipgloss.Style
	ItemFocused     lipgloss.Style
	InfoTextBlurred lipgloss.Style
	InfoTextFocused lipgloss.Style
}

func matchedRanges(in []int) [][2]int {
	if len(in) == 0 {
		return [][2]int{}
	}
	current := [2]int{in[0], in[0]}
	if len(in) == 1 {
		return [][2]int{current}
	}
	var out [][2]int
	for i := 1; i < len(in); i++ {
		if in[i] == current[1]+1 {
			current[1] = in[i]
		} else {
			out = append(out, current)
			current = [2]int{in[i], in[i]}
		}
	}
	out = append(out, current)
	return out
}

func bytePosToVisibleCharPos(str string, rng [2]int) (int, int) {
	bytePos, byteStart, byteStop := 0, rng[0], rng[1]
	pos, start, stop := 0, 0, 0
	gr := uniseg.NewGraphemes(str)
	for byteStart > bytePos {
		if !gr.Next() {
			break
		}
		bytePos += len(gr.Str())
		pos += max(1, gr.Width())
	}
	start = pos
	for byteStop > bytePos {
		if !gr.Next() {
			break
		}
		bytePos += len(gr.Str())
		pos += max(1, gr.Width())
	}
	stop = pos
	return start, stop
}
