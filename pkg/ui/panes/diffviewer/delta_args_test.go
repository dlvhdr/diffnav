package diffviewer

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/dlvhdr/diffnav/pkg/ui/common"
)

func newTestModel(t *testing.T) Model {
	t.Helper()
	common.RegisterSupportedTints()
	styles := common.MakeStyles()
	return New(ModelOpts{}, &styles)
}

func TestMakeDeltaArgs(t *testing.T) {
	m := newTestModel(t)
	want := []string{"--paging=never", "-w=80", "--wrap-max-lines=unlimited"}
	got := m.makeDeltaArgs(deltaOpts{SideBySide: true, Width: 80, UseUserDeltaTheme: true})
	if !slices.Equal(got, want) {
		t.Fatalf("user delta theme: expected %q, got %q", want, got)
	}

	got = m.makeDeltaArgs(deltaOpts{SideBySide: true, Width: 80})
	for _, arg := range []string{"--side-by-side", "--syntax-theme=", "--plus-style="} {
		if !slices.ContainsFunc(got, func(a string) bool { return strings.HasPrefix(a, arg) }) {
			t.Errorf("themed: expected %q in %q", arg, got)
		}
	}
}

func TestMakeDeltaEnv(t *testing.T) {
	t.Setenv("GIT_CONFIG_PARAMETERS", "'delta.navigate'='true'")

	if got := makeDeltaEnv(deltaOpts{SideBySide: true}); !slices.Equal(got, os.Environ()) {
		t.Fatalf("themed: expected the plain environment, got %q", got)
	}

	env := makeDeltaEnv(deltaOpts{UseUserDeltaTheme: true})
	got := env[len(env)-1]
	want := "GIT_CONFIG_PARAMETERS='delta.navigate'='true' 'delta.side-by-side'='false'"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestDiffContentMsg_IgnoresStaleResult(t *testing.T) {
	m := newTestModel(t)
	m.SetSize(80, 10)
	key := cacheKey("a.go", false)
	older := &cachedNode{path: "a.go"}
	newer := &cachedNode{path: "a.go"}
	m.file = newer
	m.cache[key] = newer

	m, _ = m.Update(diffContentMsg{cacheKey: key, node: newer, lines: stringToDiffLines("new")})
	m, _ = m.Update(diffContentMsg{cacheKey: key, node: older, lines: stringToDiffLines("old")})

	if view := m.View(); !strings.Contains(view, "new") || strings.Contains(view, "old") {
		t.Fatalf("expected only the newer result on screen, got:\n%s", view)
	}
}
