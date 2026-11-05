package ui

import (
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/bluekeyes/go-gitdiff/gitdiff"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/dlvhdr/diffnav/pkg/config"
)

func TestSearchUpdateEnterWithNoResultsDoesNotPanic(t *testing.T) {
	m := newTestMainModel(t)
	m.width = 100
	m.height = 40
	m.searchingFiles = true
	m.filesSearch.Focus()
	m.filesSearch.SetValue("does-not-match")
	m.setSearchResults()

	updated, _ := m.searchUpdate(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))

	if updated.searchingFiles {
		t.Fatal("expected search to stop after pressing enter")
	}
	if updated.resultsCursor != 0 {
		t.Fatalf("expected cursor to remain at 0, got %d", updated.resultsCursor)
	}
}

func TestSearchUpdateKeepsCursorValidWhenResultsAreEmpty(t *testing.T) {
	m := newTestMainModel(t)
	m.searchingFiles = true
	m.filesSearch.Focus()
	m.filtered = nil
	m.resultsCursor = 0

	updated, _ := m.searchUpdate(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	if updated.resultsCursor != 0 {
		t.Fatalf(
			"expected cursor to remain at 0 after down on empty results, got %d",
			updated.resultsCursor,
		)
	}

	updated.resultsCursor = -3
	updated.filesSearch.SetValue("does-not-match")
	updated.setSearchResults()
	if updated.resultsCursor != 0 {
		t.Fatalf("expected cursor to clamp to 0 for empty results, got %d", updated.resultsCursor)
	}
}

func TestSearchResultsRenderWhenFileTreeIsHidden(t *testing.T) {
	m := newTestMainModel(t)
	m.width = 100
	m.height = 40
	m.isShowingFileTree = false
	m.searchingFiles = true
	m.filesSearch.SetWidth(m.searchWidth())
	m.setSearchResults()
	m.resultsVp.SetWidth(m.config.UI.SearchTreeWidth)
	m.resultsVp.SetHeight(m.mainContentHeight() - searchHeight)
	m.resultsVp.SetContent(m.resultsView())

	view := m.View().Content
	if !strings.Contains(view, "yarn.lock") {
		t.Fatal("expected search results to be visible even when the file tree is hidden")
	}
}

func TestHiddenTreeSearchEnterThenToggleDoesNotPanic(t *testing.T) {
	m := newTestMainModel(t)
	m.width = 100
	m.height = 40

	m = updateMainModel(t, m, tea.KeyPressMsg(tea.Key{Text: "e", Code: 'e'}))
	m = updateMainModel(t, m, tea.KeyPressMsg(tea.Key{Text: "t", Code: 't'}))
	m = updateMainModel(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updateMainModel(t, m, tea.KeyPressMsg(tea.Key{Text: "e", Code: 'e'}))

	if !m.isShowingFileTree {
		t.Fatal("expected file tree to be visible after toggling it back on")
	}
	if m.filesSearch.Width() < 0 {
		t.Fatalf("expected non-negative search width, got %d", m.filesSearch.Width())
	}
	_ = m.View().Content
}

func TestHiddenTreeSearchClickNearLeftEdgeDoesNotShowFileTree(t *testing.T) {
	m := newTestMainModel(t)
	m.width = 100
	m.height = 40
	m.isShowingFileTree = false
	m.searchingFiles = true

	updated, _ := m.handleMouse(tea.MouseClickMsg(tea.Mouse{X: 1, Y: 1, Button: tea.MouseLeft}))

	result, ok := updated.(mainModel)
	if !ok {
		t.Fatalf("unexpected model type %T", updated)
	}
	if result.isShowingFileTree {
		t.Fatal("expected left-edge click during hidden-tree search to leave the file tree hidden")
	}
}

func TestHiddenSidebarGrabStillShowsFileTreeWhenNotSearching(t *testing.T) {
	m := newTestMainModel(t)
	m.width = 100
	m.height = 40
	m.isShowingFileTree = false
	m.searchingFiles = false

	updated, _ := m.handleMouse(tea.MouseClickMsg(tea.Mouse{X: 1, Y: 1, Button: tea.MouseLeft}))

	result, ok := updated.(mainModel)
	if !ok {
		t.Fatalf("unexpected model type %T", updated)
	}
	if !result.isShowingFileTree {
		t.Fatal("expected left-edge click on the hidden sidebar grab line to show the file tree")
	}
}

func TestSearchSidebarBorderClickDoesNotStartDragging(t *testing.T) {
	m := newTestMainModel(t)
	m.width = 100
	m.height = 40
	m.isShowingFileTree = true
	m.searchingFiles = true
	m.fileTree.SetSize(m.config.UI.FileTreeWidth, m.mainContentHeight()-searchHeight)

	updated, _ := m.handleMouse(tea.MouseClickMsg(tea.Mouse{
		X:      m.sidebarWidth(),
		Y:      1,
		Button: tea.MouseLeft,
	}))

	result, ok := updated.(mainModel)
	if !ok {
		t.Fatalf("unexpected model type %T", updated)
	}
	if result.draggingSidebar {
		t.Fatal("expected sidebar dragging to stay disabled while searching")
	}
}

func TestSearchSidebarDragMotionIsIgnored(t *testing.T) {
	m := newTestMainModel(t)
	m.width = 100
	m.height = 40
	m.isShowingFileTree = true
	m.searchingFiles = true
	m.draggingSidebar = true
	m.fileTree.SetSize(m.config.UI.FileTreeWidth, m.mainContentHeight()-searchHeight)

	updated, _ := m.handleMouse(tea.MouseMotionMsg(tea.Mouse{
		X:      40,
		Y:      1,
		Button: tea.MouseLeft,
	}))

	result, ok := updated.(mainModel)
	if !ok {
		t.Fatalf("unexpected model type %T", updated)
	}
	if result.draggingSidebar {
		t.Fatal("expected search-mode drag motion to clear dragging state")
	}
	if result.fileTree.Width() != m.fileTree.Width() {
		t.Fatalf(
			"expected file tree width to remain %d, got %d",
			m.fileTree.Width(),
			result.fileTree.Width(),
		)
	}
}

func TestInitialActivePanelWhenFileTreeIsHidden(t *testing.T) {
	zone.NewGlobal()

	cfg := config.DefaultConfig()
	cfg.UI.ShowFileTree = false

	data, err := os.ReadFile("../../examples/multiple_files.diff")
	if err != nil {
		t.Fatal(err)
	}

	m := New(ModelOpts{Input: string(data)}, cfg)

	if m.activePanel != DiffViewerPanel {
		t.Fatalf(
			"expected activePanel to be DiffViewerPanel when showFileTree is false, got %v",
			m.activePanel,
		)
	}
	if m.isShowingFileTree {
		t.Fatal("expected file tree to be hidden when showFileTree is false")
	}
}

func newTestMainModel(t *testing.T) mainModel {
	t.Helper()
	zone.NewGlobal()

	cfg := config.DefaultConfig()
	data, err := os.ReadFile("../../examples/multiple_files.diff")
	if err != nil {
		t.Fatal(err)
	}

	files, _, err := gitdiff.Parse(strings.NewReader(string(data) + "\n"))
	if err != nil {
		t.Fatal(err)
	}

	m := New(ModelOpts{Input: string(data)}, cfg)
	m.files = files
	m.fileTree = m.fileTree.SetFiles(files)

	return m
}

func updateMainModel(t *testing.T, m mainModel, msg tea.Msg) mainModel {
	t.Helper()

	updated, _ := m.Update(msg)
	result, ok := updated.(mainModel)
	if !ok {
		t.Fatalf("unexpected model type %T", updated)
	}

	return result
}

func typeText(t *testing.T, m mainModel, text string) mainModel {
	t.Helper()
	for _, r := range text {
		m = updateMainModel(t, m, tea.KeyPressMsg(tea.Key{Text: string(r), Code: r}))
	}
	return m
}

func applyStickyFilter(t *testing.T, m mainModel, query string) mainModel {
	t.Helper()
	m = updateMainModel(t, m, tea.KeyPressMsg(tea.Key{Text: "t", Code: 't'}))
	m = typeText(t, m, query)
	m = updateMainModel(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	m = updateMainModel(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	return m
}

func TestStickyFilterKeepsOnlyMatchingFilesInTree(t *testing.T) {
	m := newTestMainModel(t)
	m = updateMainModel(t, m, tea.WindowSizeMsg{Width: 100, Height: 40})

	m = applyStickyFilter(t, m, "package")

	if m.searchingFiles {
		t.Fatal("expected search to close after applying the sticky filter")
	}
	if m.stickyFilter != "package" {
		t.Fatalf("expected sticky filter %q, got %q", "package", m.stickyFilter)
	}
	if len(m.visibleFiles()) != 1 {
		t.Fatalf("expected 1 visible file, got %d", len(m.visibleFiles()))
	}
	tree := m.fileTree.View()
	if strings.Contains(tree, "yarn.lock") {
		t.Fatal("expected yarn.lock to be filtered out of the tree")
	}
	if !strings.Contains(tree, "package.json") {
		t.Fatal("expected package.json to be in the filtered tree")
	}
	if m.fileTree.CurrNodePath() != "graphql-server/tests/package.json" {
		t.Fatalf("expected cursor on package.json, got %q", m.fileTree.CurrNodePath())
	}

	// navigating past the only file must not escape the filter
	m = updateMainModel(t, m, tea.KeyPressMsg(tea.Key{Text: "n", Code: 'n'}))
	if m.fileTree.CurrNodePath() != "graphql-server/tests/package.json" {
		t.Fatalf("expected next file to stay within the filter, got %q", m.fileTree.CurrNodePath())
	}
	if !strings.Contains(m.View().Content, "package") {
		t.Fatal("expected the active sticky filter to be shown")
	}
}

func TestStickyFilterClearedWithEsc(t *testing.T) {
	m := newTestMainModel(t)
	m = updateMainModel(t, m, tea.WindowSizeMsg{Width: 100, Height: 40})

	m = applyStickyFilter(t, m, "package")
	m = updateMainModel(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))

	if m.stickyFilter != "" {
		t.Fatalf("expected sticky filter to be cleared, got %q", m.stickyFilter)
	}
	if !strings.Contains(m.fileTree.View(), "yarn.lock") {
		t.Fatal("expected all files to be back in the tree")
	}
	if m.fileTree.CurrNodePath() != "graphql-server/tests/package.json" {
		t.Fatalf("expected cursor to stay on package.json, got %q", m.fileTree.CurrNodePath())
	}
}

func TestSearchWithoutStickyDoesNotFilterTree(t *testing.T) {
	m := newTestMainModel(t)
	m = updateMainModel(t, m, tea.WindowSizeMsg{Width: 100, Height: 40})

	m = updateMainModel(t, m, tea.KeyPressMsg(tea.Key{Text: "t", Code: 't'}))
	m = typeText(t, m, "package")
	m = updateMainModel(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))

	if m.stickyFilter != "" {
		t.Fatalf("expected no sticky filter, got %q", m.stickyFilter)
	}
	if !strings.Contains(m.fileTree.View(), "yarn.lock") {
		t.Fatal("expected all files to stay in the tree")
	}
}

func TestStickyFilterSurvivesDiffReload(t *testing.T) {
	m := newTestMainModel(t)
	m = updateMainModel(t, m, tea.WindowSizeMsg{Width: 100, Height: 40})

	m = applyStickyFilter(t, m, "package")
	files := m.files
	m = updateMainModel(t, m, parsedDiffMsg{files: files})

	if m.stickyFilter != "package" {
		t.Fatalf("expected sticky filter to survive reload, got %q", m.stickyFilter)
	}
	if strings.Contains(m.fileTree.View(), "yarn.lock") {
		t.Fatal("expected yarn.lock to stay filtered out after reload")
	}
}
