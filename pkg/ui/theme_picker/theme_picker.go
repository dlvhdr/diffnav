package themepicker

import (
	"sort"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dlvhdr/diffnav/pkg/ui/common"
	"github.com/dlvhdr/diffnav/pkg/ui/list"
	tint "github.com/lrstanley/bubbletint/v2"
)

type Model struct {
	common.Common
	common.Component

	input         textinput.Model
	allTintsNames []string
	selected      string
	list          *list.FilterableList
}

func New(styles *common.Styles) Model {
	ti := textinput.New()
	ti.Placeholder = "Type to filter"
	ti.SetValue("")
	ti.Focus()

	lsItems := make([]list.FilterableItem, 0)
	sorted := make([]string, 0)
	for _, ti := range common.Themes.TintIDs() {
		sorted = append(sorted, ti)
	}
	sort.Strings(sorted)

	for _, ti := range sorted {
		if tn, ok := common.Themes.GetTint(ti); ok {
			lsItems = append(lsItems, NewThemeItem(styles, tn.ID, tn.DisplayName, tn))
		}
	}

	ls := list.NewFilterableList()
	ls.Focus()
	ls.SetItems(lsItems...)
	ls.SetFilter("")
	ls.ScrollToTop()
	ls.SetSelected(0)

	return Model{
		list:  ls,
		input: ti,
	}
}

type KeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
}

var Keys = &KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "ctrl+p"),
		key.WithHelp("↑/ctrl+p", "prev theme"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "ctrl+n"),
		key.WithHelp("↓/ctrl+n", "next theme"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter", "ctrl+y"),
		key.WithHelp("enter", "choose"),
	),
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, Keys.Up):
			m.list.Focus()
			if m.list.IsSelectedFirst() {
				m.list.SelectLast()
			} else {
				m.list.SelectPrev()
			}
			m.list.ScrollToSelected()
			return m.makeThemePreviewCmd()
		case key.Matches(msg, Keys.Down):
			m.list.Focus()
			if m.list.IsSelectedLast() {
				m.list.SelectFirst()
			} else {
				m.list.SelectNext()
			}
			m.list.ScrollToSelected()
			return m.makeThemePreviewCmd()
		case key.Matches(msg, Keys.Select):
			return m.makeThemeChosenCmd()
		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			value := m.input.Value()
			m.list.SetFilter(value)
			m.list.ScrollToTop()
			m.list.SetSelected(0)
			return tea.Batch(cmd, m.makeThemePreviewCmd())
		}
	}

	return nil
}

func (m *Model) View() string {
	return lipgloss.JoinVertical(lipgloss.Left, m.input.View(), m.list.Render())
}

func (m *Model) SetSize(width, height int) {
	m.Width = width
	m.Height = height
	m.input.SetWidth(m.Width)
	m.list.SetSize(m.Width, max(0, m.Height-2))
}

func (m *Model) Selected() string {
	return m.selected
}

func (m *Model) Open() tea.Cmd {
	m.input.SetValue("")
	m.list.SetFilter("")
	current := common.Themes.Current()
	if current == nil {
		return nil
	}

	for i, item := range m.list.FilteredItems() {
		if ti, ok := item.(*TintItem); ok && ti.tint.ID == current.ID {
			m.list.SetSelected(i)
			m.list.ScrollToSelected()
			return nil
		}
	}
	return nil
}

type ThemeChosenMsg struct {
	Theme *tint.Tint
}

func (m *Model) makeThemeChosenCmd() tea.Cmd {
	if selectedItem := m.list.SelectedItem(); selectedItem != nil {
		if item, ok := selectedItem.(*TintItem); ok && item != nil {
			return func() tea.Msg {
				return ThemeChosenMsg{
					Theme: item.tint,
				}
			}
		}
	}
	return nil
}

type ThemePreviewMsg struct {
	Theme *tint.Tint
}

func (m *Model) makeThemePreviewCmd() tea.Cmd {
	if selectedItem := m.list.SelectedItem(); selectedItem != nil {
		if item, ok := selectedItem.(*TintItem); ok && item != nil {
			return func() tea.Msg {
				return ThemePreviewMsg{
					Theme: item.tint,
				}
			}
		}
	}
	return nil
}
