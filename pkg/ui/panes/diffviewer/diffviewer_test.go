package diffviewer

import (
	"slices"
	"strings"
	"testing"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/charmbracelet/x/ansi"
	"github.com/dlvhdr/diffnav/pkg/ui/common"
)

func TestRenderPreamble_Empty(t *testing.T) {
	common.RegisterSupportedTints()
	m := New(false, true, &common.Styles{
		Tint:   common.Themes.Current(),
		Colors: common.Colors{},
	})
	if got := m.renderPreamble(""); got != "" {
		t.Fatalf("expected empty string for empty preamble, got %q", got)
	}
	if got := m.renderPreamble("   \n  \n  "); got != "" {
		t.Fatalf("expected empty string for whitespace-only preamble, got %q", got)
	}
}

func TestRenderPreamble_GitShow(t *testing.T) {
	common.RegisterSupportedTints()
	m := New(false, true, &common.Styles{
		Tint:   common.Themes.Current(),
		Colors: common.Colors{},
	})
	preamble := `commit abc123def456
Author: Jane Doe <jane@example.com>
Date:   Mon Jan 1 00:00:00 2026 +0000

    feat: add new feature

    This is the body of the commit message.`

	got := m.renderPreamble(preamble)
	plain := ansi.Strip(got)

	// All original content lines should be preserved in the output.
	for _, want := range []string{
		"commit abc123def456",
		"Author: Jane Doe <jane@example.com>",
		"Date:   Mon Jan 1 00:00:00 2026 +0000",
		"feat: add new feature",
		"This is the body of the commit message.",
	} {
		if !strings.Contains(plain, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, plain)
		}
	}
}

func TestRenderPreamble_MergeCommit(t *testing.T) {
	preamble := `commit abc123def456
Merge: aaa111 bbb222
Author: Jane Doe <jane@example.com>
Date:   Mon Jan 1 00:00:00 2026 +0000

    Merge branch 'feature' into main`

	common.RegisterSupportedTints()
	m := New(false, true, &common.Styles{
		Tint:   common.Themes.Current(),
		Colors: common.Colors{},
	})
	got := m.renderPreamble(preamble)
	plain := ansi.Strip(got)

	for _, want := range []string{
		"Merge: aaa111 bbb222",
		"Merge branch 'feature' into main",
	} {
		if !strings.Contains(plain, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, plain)
		}
	}
}

func TestMakeDeltaArgs_WrapLines(t *testing.T) {
	common.RegisterSupportedTints()
	styles := common.MakeStyles()

	wrapped := New(false, true, &styles)
	if args := wrapped.makeDeltaArgs(false, 80, deltaOpts{}); !slices.Contains(
		args,
		"--wrap-max-lines=unlimited",
	) {
		t.Errorf("expected wrapping on, got %v", args)
	}

	unwrapped := New(false, false, &styles)
	args := unwrapped.makeDeltaArgs(false, 80, deltaOpts{})
	if !slices.Contains(args, "--wrap-max-lines=0") {
		t.Errorf("expected wrapping off, got %v", args)
	}
	// the full line must survive so the viewport can scroll it horizontally
	if !slices.Contains(args, "--max-line-length=0") {
		t.Errorf("expected no truncation, got %v", args)
	}
}

func TestRenderSideBySide_RequiresWrap(t *testing.T) {
	common.RegisterSupportedTints()
	styles := common.MakeStyles()

	if m := New(true, false, &styles); m.renderSideBySide() {
		t.Error("side-by-side must not render when wrapping is off")
	}
	if m := New(true, true, &styles); !m.renderSideBySide() {
		t.Error("expected side-by-side when wrapping is on")
	}
}

func TestCacheKey_SeparatesWrapMode(t *testing.T) {
	wrapOn := cacheKey("file.txt", false, true)
	wrapOff := cacheKey("file.txt", false, false)
	if wrapOn == wrapOff {
		t.Errorf("expected distinct cache keys, got %q for both modes", wrapOn)
	}
}

func TestRootDiffStats_SurvivesWrapToggle(t *testing.T) {
	common.RegisterSupportedTints()
	styles := common.MakeStyles()

	files, _, err := gitdiff.Parse(strings.NewReader(
		"diff --git a/test.txt b/test.txt\n" +
			"index 1111111..2222222 100644\n" +
			"--- a/test.txt\n" +
			"+++ b/test.txt\n" +
			"@@ -1,2 +1,2 @@\n" +
			"-short\n" +
			"-short2\n" +
			"+long line\n" +
			"+long line2\n",
	))
	if err != nil {
		t.Fatal(err)
	}

	m := New(true, true, &styles)
	m, _ = m.SetDirPatch("/", files)
	m, _ = m.SetFilePatch(files[0])

	if additions, deletions := m.RootDiffStats(); additions != 2 || deletions != 2 {
		t.Errorf("expected +2/-2 root stats, got +%d/-%d", additions, deletions)
	}

	_ = m.SetWrapLines(false)
	if additions, deletions := m.RootDiffStats(); additions != 2 || deletions != 2 {
		t.Errorf("expected stats to survive wrap toggle, got +%d/-%d", additions, deletions)
	}
}

func TestSetWrapLines_TogglesViewportWrap(t *testing.T) {
	common.RegisterSupportedTints()
	styles := common.MakeStyles()

	m := New(false, true, &styles)
	_ = m.SetWrapLines(false)
	if m.fvp.GetWrapText() {
		t.Error("expected viewport wrapping off after SetWrapLines(false)")
	}
	_ = m.SetWrapLines(true)
	if !m.fvp.GetWrapText() {
		t.Error("expected viewport wrapping on after SetWrapLines(true)")
	}
}

func TestUpdate_IgnoresStaleDiffContentMsg(t *testing.T) {
	common.RegisterSupportedTints()
	styles := common.MakeStyles()

	files, _, err := gitdiff.Parse(strings.NewReader(
		"diff --git a/test.txt b/test.txt\n" +
			"index 1111111..2222222 100644\n" +
			"--- a/test.txt\n" +
			"+++ b/test.txt\n" +
			"@@ -1 +1 @@\n" +
			"-short\n" +
			"+long line\n",
	))
	if err != nil {
		t.Fatal(err)
	}

	m := New(true, true, &styles)
	m, _ = m.SetFilePatch(files[0])

	m.fvp.SetWidth(40)
	m.fvp.SetHeight(10)

	fresh := stringToDiffLines("fresh content")
	m, _ = m.Update(diffContentMsg{cacheKey: m.currentCacheKey(), lines: fresh})
	if !strings.Contains(m.fvp.View(), "fresh content") {
		t.Error("expected current-key render to be shown")
	}

	stale := stringToDiffLines("stale render that must be dropped")
	m, _ = m.Update(diffContentMsg{cacheKey: "other:node", lines: stale})
	if strings.Contains(m.fvp.View(), "stale render") {
		t.Error("stale render must not replace the displayed content")
	}
}
