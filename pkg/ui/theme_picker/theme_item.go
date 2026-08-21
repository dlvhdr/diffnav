package themepicker

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/dlvhdr/diffnav/pkg/ui/common"
	"github.com/dlvhdr/diffnav/pkg/ui/list"
	tint "github.com/lrstanley/bubbletint/v2"
	"github.com/sahilm/fuzzy"
)

// TintItem wraps a uicmd.Command to implement the ListItem interface.
type TintItem struct {
	*list.Versioned
	id       string
	title    string
	tint     *tint.Tint
	aliases  []string
	t        *common.Styles
	m        fuzzy.Match
	cache    map[int]string
	focused  bool
	hideInfo bool
}

var _ ListItem = &TintItem{Versioned: list.NewVersioned()}

func NewThemeItem(
	t *common.Styles,
	id, title string,
	tint *tint.Tint,
) *TintItem {
	bg := "light"
	if tint.Dark {
		bg = "dark"
	}
	return &TintItem{
		Versioned: list.NewVersioned(),
		id:        id,
		t:         t,
		title:     fmt.Sprintf("%s (%s)", tint.DisplayName, bg),
		tint:      tint,
	}
}

// Finished implements list.Item. Command items are render-stable
// outside of explicit SetFocused / SetMatch.
func (ti *TintItem) Finished() bool {
	return true
}

// WithAliases returns the TintItem with the given aliases for filtering.
func (ti *TintItem) WithAliases(aliases ...string) *TintItem {
	ti.aliases = aliases
	return ti
}

// Filter implements ListItem.
func (ti *TintItem) Filter() string {
	val := ti.title + " " + ti.tint.ID
	if len(ti.aliases) > 0 {
		val = val + " " + strings.Join(ti.aliases, " ")
	}
	return val
}

// ID implements ListItem.
func (ti *TintItem) ID() string {
	return ti.id
}

// SetFocused implements ListItem.
func (ti *TintItem) SetFocused(focused bool) {
	if ti.focused == focused {
		return
	}
	ti.cache = nil
	ti.focused = focused
	if ti.Versioned != nil {
		ti.Bump()
	}
}

// SetMatch implements ListItem.
func (ti *TintItem) SetMatch(m fuzzy.Match) {
	if sameFuzzyMatch(ti.m, m) {
		return
	}
	ti.cache = nil
	ti.m = m
	if ti.Versioned != nil {
		ti.Bump()
	}
}

func (ti *TintItem) Tint() *tint.Tint {
	return ti.tint
}

// SetHideInfo controls whether the hint column is shown. The
// dialog hides it uniformly when it would crowd the command names.
func (ti *TintItem) SetHideInfo(v bool) {
	if ti.hideInfo == v {
		return
	}
	ti.cache = nil
	ti.hideInfo = v
	if ti.Versioned != nil {
		ti.Bump()
	}
}

// Render implements ListItem.
func (ti *TintItem) Render(width int) string {
	selectedBgStyle := lipgloss.NewStyle().
		Background(ti.t.Colors.SelectionBg)
	styles := ListItemStyles{
		ItemBlurred: lipgloss.NewStyle().PaddingLeft(1),
		ItemFocused: selectedBgStyle.PaddingLeft(1).Foreground(ti.t.Tint.BrightWhite).
			Bold(true),
		InfoTextBlurred: lipgloss.NewStyle().PaddingLeft(1).Foreground(ti.t.Tint.White),
		InfoTextFocused: selectedBgStyle.PaddingLeft(1).Foreground(ti.t.Tint.White),
	}

	var bg color.Color
	descStyle := styles.InfoTextBlurred
	if ti.focused {
		bg = selectedBgStyle.GetBackground()
		descStyle = styles.InfoTextFocused
	}

	id := ti.tint.ID
	id = ansi.Truncate(strings.TrimSpace(id), width-1, "...")
	descVisWidth := lipgloss.Width(id)
	gap := strings.Repeat(" ", max(0, width-descVisWidth-1))
	if id == "" {
		id = " "
	}
	rendered := renderItem(
		styles,
		ti.title,
		themeSwatchStrip(ti.tint.ID, bg),
		ti.focused,
		width,
		ti.cache,
		&ti.m,
	)
	return lipgloss.JoinVertical(lipgloss.Left, rendered, descStyle.Render(id+gap))
}

// ThemeSwatch returns a small, representative set of colors for a theme id, for
// previewing it in the theme picker. Unknown or empty ids return the default
// palette.
func themeSwatch(id string) []color.Color {
	t, ok := common.Themes.GetTint(id)
	if !ok || t == nil {
		return []color.Color{}
	}
	return []color.Color{
		t.BrightRed, t.BrightYellow, t.BrightGreen,
		t.BrightCyan, t.BrightBlue, t.BrightPurple,
		t.Fg, t.Bg,
	}
}

// themeSwatchStrip renders a theme's preview colors as adjacent two-cell blocks.
func themeSwatchStrip(id string, bg color.Color) string {
	colors := themeSwatch(id)
	var b strings.Builder
	for _, c := range colors {
		b.WriteString(lipgloss.NewStyle().Background(c).Render("  "))
	}
	// A trailing surface cell separates the strip from the panel edge cleanly.
	return b.String() + lipgloss.NewStyle().Background(bg).Render(" ")
}
