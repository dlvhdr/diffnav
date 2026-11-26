package diffviewer

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/log/v2"
	"github.com/bluekeyes/go-gitdiff/gitdiff"

	"github.com/dlvhdr/diffnav/pkg/filenode"
	"github.com/dlvhdr/diffnav/pkg/icons"
	"github.com/dlvhdr/diffnav/pkg/ui/common"
	"github.com/dlvhdr/diffnav/pkg/utils"
	"github.com/robinovitch61/viewport/filterableviewport"
	"github.com/robinovitch61/viewport/viewport"
	"github.com/robinovitch61/viewport/viewport/item"
)

const dirHeaderHeight = 3

type cachedNode struct {
	path      string
	files     []*gitdiff.File
	additions int64
	deletions int64
	diff      []diffLine
}

type nodeCache map[string]*cachedNode

func cacheKey(path string, sideBySide bool) string {
	key := path + ":" + common.Themes.Current().ID
	if sideBySide {
		return key + ":sbs"
	}
	return key
}

type diffLine struct {
	item    item.Item
	records []record
}

func (o diffLine) GetItem() item.Item {
	return o.item
}

var ViewportKeyMap = viewport.KeyMap{
	HalfPageDown: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", "scroll half page down"),
	),
	HalfPageUp: key.NewBinding(
		key.WithKeys("ctrl+u"),
		key.WithHelp("ctrl+u", "scroll half page up"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k", "ctrl+y"),
		key.WithHelp("↑/k", "prev file"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j", "ctrl+e"),
		key.WithHelp("↓/j", "next file"),
	),
	Bottom: key.NewBinding(
		key.WithKeys("G"),
		key.WithHelp("G", "bottom"),
	),
	Top: key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("g", "top"),
	),
	Left: key.NewBinding(
		key.WithKeys("left"),
		key.WithHelp("←", "scroll left"),
	),
	Right: key.NewBinding(
		key.WithKeys("right"),
		key.WithHelp("→", "scroll right"),
	),
}

type Model struct {
	common.Common
	fvp        *filterableviewport.Model[diffLine]
	file       *cachedNode
	dir        *cachedNode
	cache      nodeCache
	sideBySide bool
	// useUserDeltaTheme renders diffs with the user's own delta config instead of the theme.
	useUserDeltaTheme bool
	preamble          string
	sb                common.Scrollbar
	// shown are the lines on screen: the last render while a new one is pending.
	shown []diffLine
	// anchor is where to scroll back to once the re-rendered diff is shown.
	anchor *scrollAnchor
}

// scrollAnchor is where to scroll back to after a re-render: the top or bottom of the
// diff, or a diff line and the view's offset from its first row.
type scrollAnchor struct {
	edge   anchorEdge
	rec    record
	offset int
}

// anchorEdge is the edge of the diff an anchor sticks to, if any.
type anchorEdge int

const (
	edgeNone anchorEdge = iota
	edgeTop
	edgeBottom
)

// SetPreamble stores the preamble text (e.g. commit metadata from git show).
func (m *Model) SetPreamble(preamble string) {
	m.preamble = preamble
}

type ModelOpts struct {
	SideBySide        bool
	UseUserDeltaTheme bool
}

func New(opts ModelOpts, styles *common.Styles) Model {
	vp := viewport.New(
		0,
		0,
		viewport.WithKeyMap[diffLine](ViewportKeyMap),
		viewport.WithStyles[diffLine](
			viewport.Styles{
				SelectionPrefix: lipgloss.NewStyle().Foreground(styles.Tint.Blue).Render("▐"),
				SelectedItemStyle: lipgloss.NewStyle().
					Background(styles.Colors.SelectionBg).
					Foreground(styles.Tint.BrightWhite),
				FooterStyle: lipgloss.NewStyle().
					BorderForeground(styles.Tint.BrightBlack).
					Background(styles.Tint.BrightBlack).
					Border(lipgloss.Border{Left: "", Right: ""}, false, true, false, true).
					Foreground(styles.Tint.White).
					Italic(true),
			},
		),
	)

	sb := common.Scrollbar{
		Styles: common.ScrollbarStyles{
			Thumb: lipgloss.NewStyle().Foreground(styles.Tint.Blue),
			Track: lipgloss.NewStyle().Foreground(styles.Tint.Black),
		},
	}

	filterableViewportKeyMap := filterableviewport.DefaultKeyMap()
	filterableViewportKeyMap.CancelFilterKey = key.NewBinding(
		key.WithKeys("esc", "ctrl+c"),
		key.WithHelp("esc/ctrl+c", "cancel filter"),
	)

	filterableViewportStyles := filterableviewport.DefaultStyles()
	filterableViewportStyles.Filter.Focused.TextInput.Text = lipgloss.NewStyle().
		Foreground(styles.Tint.BrightWhite)
	filterableViewportStyles.Filter.Unfocused.TextInput.Text = lipgloss.NewStyle().
		Foreground(styles.Tint.White)
	filterableViewportStyles.Filter.Empty = lipgloss.NewStyle().Foreground(styles.Tint.White)
	filterableViewportStyles.MatchesCount.Matches = lipgloss.NewStyle().
		Foreground(styles.Tint.BrightYellow)
	filterableViewportStyles.Match.Focused = lipgloss.NewStyle().
		Background(lipgloss.Darken(styles.Tint.Yellow, 0.2)).
		Foreground(lipgloss.Lighten(styles.Tint.Yellow, 0.5))
	filterableViewportStyles.Match.Unfocused = lipgloss.NewStyle().
		Background(lipgloss.Lighten(styles.Colors.SelectionBg, 0.1)).
		Foreground(lipgloss.Lighten(styles.Tint.BrightWhite, 0.5))

	return Model{
		Common: common.Common{
			Styles: styles,
		},
		sb: sb,
		fvp: filterableviewport.New(
			vp,
			filterableviewport.WithKeyMap[diffLine](filterableViewportKeyMap),
			filterableviewport.WithStyles[diffLine](filterableViewportStyles),
			filterableviewport.WithPrefixText[diffLine](
				lipgloss.NewStyle().Bold(true).Render("Filter:"),
			),
			filterableviewport.WithFilterModes[diffLine](
				[]filterableviewport.FilterMode{
					filterableviewport.CaseInsensitiveFilterMode(
						key.NewBinding(
							key.WithKeys("/"),
							key.WithHelp("/", "case insensitive"),
						),
					), filterableviewport.RegexFilterMode(
						key.NewBinding(
							key.WithKeys("ctrl+r"),
							key.WithHelp("ctrl+r", "regex fitler mode"),
						),
					),
					filterableviewport.FuzzyFilterMode(
						key.NewBinding(
							key.WithKeys("ctrl+f"),
							key.WithHelp("ctrl+f", "fuzzy fitler mode"),
						),
					),
					filterableviewport.ExactFilterMode(key.NewBinding(
						key.WithKeys("ctrl+s"),
						key.WithHelp("ctrl+s", "exact"),
					)),
				},
			),
			filterableviewport.WithPlaceholderText[diffLine]("type to search…"),
			filterableviewport.WithItemDescriptor[diffLine]("lines"),
			filterableviewport.WithEmptyText[diffLine](
				"Search… "+lipgloss.NewStyle().
					Faint(true).
					Render("(/ insensitive ⋅ ⌃+s exact ⋅ ⌃+r regex ⋅ ⌃+f fuzzy)"),
			),
			filterableviewport.WithFilterLinePosition[diffLine](filterableviewport.FilterLineTop),
			filterableviewport.WithMatchingItemsOnly[diffLine](false),
			filterableviewport.WithCanToggleMatchingItemsOnly[diffLine](true),
			filterableviewport.WithVerticalPad[diffLine](8),
			filterableviewport.WithHorizontalPad[diffLine](8),
		),
		sideBySide:        opts.SideBySide,
		useUserDeltaTheme: opts.UseUserDeltaTheme,
		cache:             map[string]*cachedNode{},
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	log.Debug("diffviewer", "msg", fmt.Sprintf("%T", msg))
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case diffContentMsg:
		node, ok := m.cache[msg.cacheKey]
		if !ok || node != msg.node {
			// Stale result from a replaced node.
			break
		}
		node.diff = msg.lines
		if node == m.file || node == m.dir {
			m.setLines(msg.lines)
		}
	}

	edge := edgeNone
	if k, ok := msg.(tea.KeyPressMsg); ok && !m.fvp.FilterFocused() {
		if key.Matches(k, ViewportKeyMap.Top) {
			edge = edgeTop
		} else if key.Matches(k, ViewportKeyMap.Bottom) {
			edge = edgeBottom
		}
	}
	row := m.currentRow()
	fvp, fvpCmd := m.fvp.Update(msg)
	cmds = append(cmds, fvpCmd)
	m.fvp = fvp
	m.reanchor(row, edge)

	return m, tea.Batch(cmds...)
}

const scrollbarWidth = 2 // 1 scrollbar character + 1 padding

func (m Model) View() string {
	vpView := m.fvp.View()
	itemMetrics := m.fvp.GetItemMetrics()
	scrollbar := m.sb.View(
		m.fvp.GetHeight(),
		itemMetrics.TotalItems,
		itemMetrics.FirstVisibleItemIdx,
		itemMetrics.LastVisibleItemIdx,
	)
	if scrollbar != "" {
		vpView = lipgloss.JoinHorizontal(lipgloss.Top, vpView, scrollbar)
	}
	return vpView
}

func (m *Model) SetSize(width, height int) tea.Cmd {
	// Before resizing: the viewport clamps its top row to the new size.
	m.keepPosition()
	m.Width = width
	m.Height = height
	m.fvp.SetWidth(m.contentWidth())
	m.fvp.SetHeight(m.Height)
	m.ClearCache()
	return m.diff()
}

func (m *Model) UpdateTheme() tea.Cmd {
	m.keepPosition()
	m.ClearCache()
	return m.diff()
}

// keepPosition remembers the diff line at the top of the view (or the selected one) and
// the view's offset from its first row, so the next render scrolls back to it (anchorRow).
func (m *Model) keepPosition() {
	// Without a record, a pending anchor is kept.
	if rec, ok := m.currentRecord(); ok {
		m.anchor = &scrollAnchor{rec: rec, offset: m.currentRow() - findRecord(m.shown, rec)}
	}
}

// currentRecord returns the first record at or below the current row of the lines on
// screen, so it is right while a new render is pending too.
func (m *Model) currentRecord() (record, bool) {
	row := m.currentRow()
	if row < 0 || m.fvp.GetMatchingItemsOnly() {
		return record{}, false
	}
	return firstRecord(m.shown, row)
}

// currentRow is the selected row, or the top visible one without a selection.
func (m *Model) currentRow() int {
	if m.SelectionEnabled() {
		return m.fvp.GetSelectedItemIdx()
	}
	return m.fvp.GetItemMetrics().FirstVisibleItemIdx
}

// reanchor keeps a pending render from undoing a move the user made from row: a jump to
// the top or bottom (edge) stays there, other moves keep the line they land on.
func (m *Model) reanchor(row int, edge anchorEdge) {
	if m.anchor == nil || m.currentRow() == row {
		return
	}
	if edge != edgeNone {
		m.anchor = &scrollAnchor{edge: edge}
		return
	}
	m.anchor = nil
	m.keepPosition()
}

// setLines shows lines and scrolls to the pending anchor, if any.
func (m *Model) setLines(lines []diffLine) {
	m.shown = lines
	m.fvp.SetObjects(lines)
	if m.anchor == nil {
		return
	}
	a := *m.anchor
	m.anchor = nil
	switch a.edge {
	case edgeTop:
		m.fvp.GoToTop()
		return
	case edgeBottom:
		m.fvp.GoToBottom()
		return
	}
	if m.fvp.GetMatchingItemsOnly() {
		return
	}
	row, ok := anchorRow(lines, a.rec, a.offset)
	if !ok {
		return
	}
	if m.SelectionEnabled() {
		m.fvp.SetSelectedItemIdx(row)
		return
	}
	m.fvp.GoToTop()
	m.fvp.ScrollDown(row)
}

// anchorRow returns the row offset rows from rec's first row, or false when rec is gone.
// Rows above it (offset < 0) have no record, like blank or decoration rows. Rows below it
// are its wrapped rows, so it stops at the line's last row.
func anchorRow(lines []diffLine, rec record, offset int) (int, bool) {
	row := findRecord(lines, rec)
	if row < 0 {
		return 0, false
	}
	if offset < 0 {
		return max(row+offset, 0), true
	}
	for range offset {
		if row+1 == len(lines) || !lines[row+1].shows(rec) {
			break
		}
		row++
	}
	return row, true
}

func (m Model) contentWidth() int {
	return m.Width - scrollbarWidth
}

func (m *Model) diff() tea.Cmd {
	if m.file != nil {
		file := m.file.files[0]
		sideBySide := m.sideBySide && !file.IsNew && !file.IsDelete
		key := cacheKey(m.file.path, sideBySide)
		if cached, ok := m.cache[key]; ok && len(cached.diff) != 0 {
			m.file = cached
			m.setLines(cached.diff)
			return nil
		}
		node := &cachedNode{
			path:      m.file.path,
			files:     m.file.files,
			additions: m.file.additions,
			deletions: m.file.deletions,
		}
		m.file = node
		m.cache[key] = node
		return m.diffFile(node, m.contentWidth())
	} else if m.dir != nil {
		key := cacheKey(m.dir.path, m.sideBySide)
		if cached, ok := m.cache[key]; ok && len(cached.diff) != 0 {
			m.dir = cached
			m.setLines(cached.diff)
			return nil
		}
		node := &cachedNode{
			path:      m.dir.path,
			files:     m.dir.files,
			additions: m.dir.additions,
			deletions: m.dir.deletions,
		}
		m.dir = node
		m.cache[key] = node
		preamble := ""
		if m.dir.path == "/" {
			preamble = m.preamble
		}
		return m.diffDir(node, m.contentWidth(), m.sideBySide, preamble)
	}

	return nil
}

func (m Model) headerView() string {
	if m.dir != nil {
		return m.dirHeaderView()
	}

	if m.file == nil || len(m.file.files) != 1 {
		return ""
	}
	name := m.file.path
	base := lipgloss.NewStyle()

	fileIcon := icons.GetIcon(name, false)
	prefix := base.Render(fileIcon) + base.Render(" ")
	name = utils.TruncateString(name, m.contentWidth()-lipgloss.Width(prefix))
	top := prefix + base.Bold(true).Render(name)

	bottom := filenode.ViewFileDiffStats(m.file.files[0], base)

	return base.
		Width(m.contentWidth()).
		Height(dirHeaderHeight - 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(m.Styles.Tint.Black).
		Render(lipgloss.JoinVertical(lipgloss.Left, top, bottom))
}

func (m Model) dirHeaderView() string {
	base := lipgloss.NewStyle().Foreground(m.Styles.Tint.Blue)
	prefix := base.Render(" ")
	name := utils.TruncateString(m.dir.path, m.contentWidth()-lipgloss.Width(prefix))

	top := prefix + base.Bold(true).Render(name)
	bottom := filenode.ViewDiffStats(m.dir.additions, m.dir.deletions, base)
	return base.
		Width(m.contentWidth()).
		Height(dirHeaderHeight - 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(m.Styles.Tint.Black).
		Render(lipgloss.JoinVertical(lipgloss.Left, top, bottom))
}

func (m Model) SetFilePatch(file *gitdiff.File) (Model, tea.Cmd) {
	m.dir = nil
	// The rows on screen are another node's until this one is shown.
	m.anchor, m.shown = nil, nil

	fname := filenode.GetFileName(file)
	sideBySide := m.sideBySide && !file.IsNew && !file.IsDelete
	key := cacheKey(fname, sideBySide)
	if cached, ok := m.cache[key]; ok {
		m.file = cached
		m.updateHeader()
		m.setLines(cached.diff)
		return m, nil
	}

	files := make([]*gitdiff.File, 1)
	files[0] = file
	additions, deletions := filenode.DiffStats(file)
	m.file = &cachedNode{
		path:      fname,
		files:     files,
		additions: additions,
		deletions: deletions,
	}
	m.cache[key] = m.file

	m.updateHeader()
	return m, m.diffFile(m.file, m.contentWidth())
}

func (m Model) SetDirPatch(dirPath string, files []*gitdiff.File) (Model, tea.Cmd) {
	m.file = nil
	m.anchor, m.shown = nil, nil

	key := cacheKey(dirPath, m.sideBySide)
	if cached, ok := m.cache[key]; ok {
		m.dir = cached
		m.updateHeader()
		m.setLines(cached.diff)
		return m, nil
	}

	var added, deleted int64
	for _, file := range files {
		na, nd := filenode.DiffStats(file)
		added += na
		deleted += nd
	}
	m.dir = &cachedNode{
		path:      dirPath,
		files:     files,
		additions: added,
		deletions: deleted,
	}
	m.cache[key] = m.dir
	preamble := ""
	if dirPath == "/" {
		preamble = m.preamble
	}
	m.updateHeader()
	return m, m.diffDir(m.dir, m.contentWidth(), m.sideBySide, preamble)
}

func (m *Model) updateHeader() {
	m.fvp.SetHeader(strings.Split(m.headerView(), "\n"))
}

// SetSideBySide updates the diff view mode and re-renders.
func (m *Model) SetSideBySide(sideBySide bool) tea.Cmd {
	m.sideBySide = sideBySide
	m.keepPosition()
	return m.diff()
}

// ScrollUp scrolls the viewport up by the given number of lines.
func (m *Model) ScrollUp(lines int) {
	row := m.currentRow()
	m.fvp.ScrollUp(lines)
	m.reanchor(row, edgeNone)
}

// ScrollDown scrolls the viewport down by the given number of lines.
func (m *Model) ScrollDown(lines int) {
	row := m.currentRow()
	m.fvp.ScrollDown(lines)
	m.reanchor(row, edgeNone)
}

// ScrollBottom scrolls the viewport to the bottom.
func (m *Model) ScrollBottom() {
	m.fvp.GoToBottom()
}

// ScrollTop scrolls the viewport to its top.
func (m *Model) ScrollTop() {
	m.fvp.GoToTop()
}

// ScrollLeft moves the viewport to the left by the given number of columns.
func (m *Model) ScrollLeft(cols int) {
	m.fvp.ScrollLeft(cols)
}

// ScrollRight scrolls the viewport one column away from column 0.
func (m *Model) ScrollRight(cols int) {
	m.fvp.ScrollRight(cols)
}

func (m *Model) diffFile(node *cachedNode, width int) tea.Cmd {
	if width == 0 || node == nil || len(node.files) != 1 {
		return nil
	}

	file := node.files[0]
	sideBySide := m.sideBySide && !file.IsNew && !file.IsDelete
	key := cacheKey(node.path, sideBySide)
	opts := deltaOpts{
		SideBySide:        sideBySide,
		Width:             width,
		IsNew:             file.IsNew,
		IsDelete:          file.IsDelete,
		UseUserDeltaTheme: m.useUserDeltaTheme,
	}
	args := m.makeDeltaArgs(opts)
	env := makeDeltaEnv(opts)
	return func() tea.Msg {
		log.Info("executing delta", "cmd", fmt.Sprintf(`delta %s`, strings.Join(args, " ")))
		deltac := exec.Command("delta", args...)
		deltac.Env = env
		deltac.Stdin = strings.NewReader(file.String() + "\n")
		out, err := deltac.Output()
		if err != nil {
			return common.ErrMsg{Err: err}
		}

		return diffContentMsg{cacheKey: key, node: node, lines: parseDeltaOutput(string(out))}
	}
}

func (m *Model) diffDir(dir *cachedNode, width int, sideBySide bool, preamble string) tea.Cmd {
	if width == 0 || dir == nil {
		return nil
	}
	key := cacheKey(dir.path, sideBySide)
	opts := deltaOpts{
		SideBySide:        sideBySide,
		Width:             width,
		UseUserDeltaTheme: m.useUserDeltaTheme,
	}
	args := m.makeDeltaArgs(opts)
	env := makeDeltaEnv(opts)
	return func() tea.Msg {
		log.Info("executing delta", "cmd", fmt.Sprintf(`delta %s`, strings.Join(args, " ")))
		deltac := exec.Command("delta", args...)
		deltac.Env = env
		strs := strings.Builder{}
		for _, file := range dir.files {
			strs.WriteString(file.String())
		}
		deltac.Stdin = strings.NewReader(strs.String() + "\n")
		out, err := deltac.Output()
		if err != nil {
			if stdErr, ok := err.(*exec.ExitError); ok {
				return common.ErrMsg{Err: err, Message: string(stdErr.Stderr)}
			}
			log.Warn("delta error", "err", err)
			return common.ErrMsg{Err: err}
		}

		lines := parseDeltaOutput(string(out))
		if preamble != "" {
			lines = append(stringToDiffLines(m.renderPreamble(preamble)), lines...)
		}
		return diffContentMsg{cacheKey: key, node: dir, lines: lines}
	}
}

func (m Model) renderPreamble(preamble string) string {
	preamble = strings.TrimSpace(preamble)
	if preamble == "" {
		return ""
	}

	dim := lipgloss.NewStyle().Foreground(m.Styles.Tint.Black)
	yellow := lipgloss.NewStyle().Foreground(m.Styles.Tint.Yellow)

	var out []string
	for line := range strings.SplitSeq(preamble, "\n") {
		switch {
		case strings.HasPrefix(line, "commit "):
			out = append(
				out,
				dim.Render("commit ")+yellow.Render(strings.TrimPrefix(line, "commit ")),
			)
		case strings.HasPrefix(line, "Author:"),
			strings.HasPrefix(line, "AuthorDate:"),
			strings.HasPrefix(line, "Date:"),
			strings.HasPrefix(line, "Commit:"),
			strings.HasPrefix(line, "CommitDate:"),
			strings.HasPrefix(line, "Merge:"):
			out = append(out, dim.Render(line))
		default:
			out = append(out, line)
		}
	}

	return strings.Join(out, "\n")
}

type diffContentMsg struct {
	cacheKey string
	node     *cachedNode
	lines    []diffLine
}

func (m *Model) ClearCache() {
	m.cache = make(nodeCache)
}

func (m *Model) RootDiffStats() (int64, int64) {
	if item, ok := m.cache[cacheKey("/", m.sideBySide)]; ok {
		return item.additions, item.deletions
	}

	return 0, 0
}

func (m *Model) Searching() bool {
	return m.fvp.FilterFocused()
}

func (m *Model) SetFiltering() {
	var mode filterableviewport.FilterModeName
	if curr := m.fvp.GetActiveFilterMode(); curr != nil {
		mode = curr.Name
	} else {
		mode = filterableviewport.FilterExact
	}
	m.fvp.SetFilter(m.fvp.GetFilterText(), mode)
}

func (m *Model) Filtering() bool {
	return m.fvp.GetActiveFilterMode() != nil
}

func (m *Model) SelectionEnabled() bool {
	return m.fvp.GetSelectionEnabled()
}

func (m *Model) SetSelectionEnabled(val bool) {
	m.fvp.SetSelectionEnabled(val)
}

func stringToDiffLines(val string) []diffLine {
	lines := strings.Split(val, "\n")
	objects := make([]diffLine, len(lines))
	for i, line := range lines {
		objects[i] = diffLine{item: item.NewItem(line)}
	}
	return objects
}

type deltaOpts struct {
	IsNew             bool
	IsDelete          bool
	SideBySide        bool
	Width             int
	UseUserDeltaTheme bool
}

// makeDeltaEnv asks for OSC 1717 line metadata and, with the user's theme, sets side-by-side
// like `git -c delta.side-by-side=...` would. The --side-by-side flag would override the
// user's line-number format and can't turn it off.
func makeDeltaEnv(opts deltaOpts) []string {
	env := append(os.Environ(), "OSC1717=V1")
	if !opts.UseUserDeltaTheme {
		return env
	}
	param := fmt.Sprintf("'delta.side-by-side'='%t'", opts.SideBySide)
	if prev := os.Getenv("GIT_CONFIG_PARAMETERS"); prev != "" {
		param = prev + " " + param
	}
	return append(env, "GIT_CONFIG_PARAMETERS="+param)
}

func (m *Model) makeDeltaArgs(opts deltaOpts) []string {
	if opts.UseUserDeltaTheme {
		// Side-by-side is set in makeDeltaEnv. Unlimited wrap so long lines aren't cut.
		return []string{
			"--paging=never",
			fmt.Sprintf("-w=%d", opts.Width),
			"--wrap-max-lines=unlimited",
		}
	}

	theme := common.Themes.Current()
	sSelection := lipgloss.NewStyle().Background(m.Styles.Colors.SelectionBg)
	selectionColor := common.LipglossColorToHex(m.Styles.Colors.SelectionBg)
	white := common.LipglossColorToHex(theme.White)
	green := common.LipglossColorToHex(lipgloss.Darken(theme.Green, 0.8))
	brightGreen := common.LipglossColorToHex(
		lipgloss.Darken(theme.BrightGreen, 0.5),
	)
	plusLine := common.LipglossColorToHex(theme.BrightGreen)
	minusLine := common.LipglossColorToHex(theme.BrightRed)
	red := common.LipglossColorToHex(lipgloss.Darken(theme.Red, 0.8))
	brightRed := common.LipglossColorToHex(lipgloss.Darken(theme.BrightRed, 0.5))
	args := []string{
		"--paging=never",
		"--line-numbers",
		"--tabs=2",
		"--wrap-left-symbol= ",
		"--wrap-right-symbol= ",
		"--wrap-right-prefix-symbol= ",
		"--hunk-label=  󰡏 ",
		fmt.Sprintf(
			"--syntax-theme=%s",
			common.SupportedThemeToDeltaSyntax[theme.ID],
		),
		fmt.Sprintf("--file-modified-label=%s",
			utils.RemoveReset(sSelection.Foreground(m.Styles.Tint.BrightYellow).Render(" "))),
		fmt.Sprintf("--file-removed-label=%s",
			utils.RemoveReset(sSelection.Foreground(m.Styles.Tint.BrightRed).Render(" "))),
		fmt.Sprintf("--file-added-label=%s",
			utils.RemoveReset(sSelection.Foreground(m.Styles.Tint.BrightGreen).Render(" "))),
		fmt.Sprintf("--file-renamed-label=%s",
			utils.RemoveReset(sSelection.Foreground(m.Styles.Tint.Blue).Render("renamed:"))),
		fmt.Sprintf("--file-style='\"%s\" bold \"%s\"'", selectionColor, selectionColor),
		fmt.Sprintf("--file-decoration-style='\"%s\" box %s'", selectionColor, selectionColor),
		fmt.Sprintf("-w=%d", opts.Width),
		"--max-line-length=0",
		"--wrap-max-lines=unlimited",
		fmt.Sprintf(
			"--line-numbers-right-style='\"%s\" dim'",
			common.LipglossColorToHex(m.Styles.Colors.FaintBlue()),
		),
		fmt.Sprintf("--line-numbers-zero-style='\"%s\" dim'", white),
		fmt.Sprintf("--plus-style='syntax \"%s\"'", green),
		fmt.Sprintf("--plus-emph-style='syntax \"%s\"'", brightGreen),
		fmt.Sprintf("--minus-style='syntax \"%s\"'", red),
		fmt.Sprintf("--minus-emph-style='syntax \"%s\"'", brightRed),
		fmt.Sprintf(
			"--hunk-header-line-number-style='%s'",
			common.LipglossColorToHex(m.Styles.Colors.FaintBlue()),
		),
		fmt.Sprintf(
			"--hunk-header-style='%s'",
			white,
		),
		fmt.Sprintf("--hunk-header-file-style='\"%s\" dim'", white),
		fmt.Sprintf(
			"--hunk-header-decoration-style='\"%s\" ol ul'",
			common.LipglossColorToHex(m.Styles.Colors.FaintBlue()),
		),
	}

	if opts.IsNew {
		args = append(
			args,
			fmt.Sprintf("--line-numbers-plus-style='\"%s\" \"%s\"'", plusLine, green),
			"--line-numbers-minus-style=",
			"--line-numbers-left-format=",
			"--line-numbers-right-format={np:>4} ",
		)
	} else if opts.IsDelete {
		args = append(
			args,
			"--line-numbers-plus-style=",
			fmt.Sprintf("--line-numbers-minus-style=\"%s\" \"%s\"", minusLine, red),
			"--line-numbers-left-format={nm:>4} ",
			"--line-numbers-right-format=",
		)
	} else {
		args = append(
			args,
			fmt.Sprintf("--line-numbers-plus-style='\"%s\" \"%s\"'", plusLine, green),
			fmt.Sprintf("--line-numbers-minus-style='\"%s\" \"%s\"", minusLine, red),
			"--line-numbers-left-format={nm:>4} ",
			"--line-numbers-right-format={np:>4} ",
		)
	}

	if opts.SideBySide {
		args = append(args, "--side-by-side")
	}
	if theme.Dark {
		args = append(args, "--dark")
	} else {
		args = append(args, "--light")
	}

	return args
}
