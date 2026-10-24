package diffviewer

import (
	"slices"
	"strings"
	"testing"

	"github.com/dlvhdr/diffnav/pkg/ui/common"
)

func newTestModel(t *testing.T, ownDelta bool) Model {
	t.Helper()
	common.RegisterSupportedTints()
	common.DeltaOwnConfig = ownDelta
	t.Cleanup(func() { common.DeltaOwnConfig = false })
	styles := common.MakeStyles()
	return New(false, &styles)
}

func hasArgPrefix(args []string, prefix string) bool {
	return slices.ContainsFunc(args, func(a string) bool { return strings.HasPrefix(a, prefix) })
}

func TestMakeDeltaArgs_OwnConfigOnlyPassesLayoutFlags(t *testing.T) {
	m := newTestModel(t, true)

	got := m.makeDeltaArgs(false, 80, deltaOpts{})
	want := []string{"--paging=never", "-w=80", "--max-line-length=0", "--wrap-max-lines=unlimited"}
	if !slices.Equal(got, want) {
		t.Fatalf("expected %q, got %q", want, got)
	}

	got = m.makeDeltaArgs(true, 80, deltaOpts{})
	if !slices.Contains(got, "--side-by-side") {
		t.Fatalf("expected --side-by-side in %q", got)
	}
}

func TestMakeDeltaArgs_ThemedPassesStyleFlags(t *testing.T) {
	m := newTestModel(t, false)

	got := m.makeDeltaArgs(false, 80, deltaOpts{})
	for _, prefix := range []string{"--syntax-theme=", "--plus-style=", "--minus-style="} {
		if !hasArgPrefix(got, prefix) {
			t.Errorf("expected an arg starting with %q in %q", prefix, got)
		}
	}
}

func TestCacheKey_DiffersByDeltaMode(t *testing.T) {
	common.RegisterSupportedTints()
	t.Cleanup(func() { common.DeltaOwnConfig = false })

	common.DeltaOwnConfig = false
	themed := cacheKey("a.go", false)
	common.DeltaOwnConfig = true
	own := cacheKey("a.go", false)

	if themed == own {
		t.Fatalf("expected different cache keys, both were %q", themed)
	}
}

func TestDiffContentMsg_IgnoresStaleResults(t *testing.T) {
	m := newTestModel(t, false)
	m.SetSize(80, 10)
	current := &cachedNode{path: "current.go"}
	other := &cachedNode{path: "other.go"}
	m.file = current
	m.cache[cacheKey(current.path, false)] = current
	m.cache[cacheKey(other.path, false)] = other

	m, _ = m.Update(diffContentMsg{cacheKey: "gone.go", lines: stringToDiffLines("stale output")})
	m, _ = m.Update(diffContentMsg{cacheKey: cacheKey(other.path, false), lines: stringToDiffLines("other output")})
	if view := m.View(); strings.Contains(view, "stale output") || strings.Contains(view, "other output") {
		t.Fatalf("expected stale and non-current results to stay off screen, got:\n%s", view)
	}
	if len(other.diff) == 0 {
		t.Fatal("expected the non-current result to still be cached")
	}

	m, _ = m.Update(diffContentMsg{cacheKey: cacheKey(current.path, false), lines: stringToDiffLines("current output")})
	if view := m.View(); !strings.Contains(view, "current output") {
		t.Fatalf("expected current result on screen, got:\n%s", view)
	}
}
