package diffviewer

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/dlvhdr/diffnav/pkg/ui/common"
)

func newTestModel(t *testing.T, useUserDeltaTheme bool) Model {
	t.Helper()
	common.RegisterSupportedTints()
	common.UseUserDeltaTheme = useUserDeltaTheme
	t.Cleanup(func() { common.UseUserDeltaTheme = false })
	styles := common.MakeStyles()
	return New(false, &styles)
}

func TestMakeDeltaArgs(t *testing.T) {
	m := newTestModel(t, true)
	want := []string{"--paging=never", "-w=80", "--wrap-max-lines=unlimited"}
	if got := m.makeDeltaArgs(true, 80, deltaOpts{}); !slices.Equal(got, want) {
		t.Fatalf("user delta theme: expected %q, got %q", want, got)
	}

	m = newTestModel(t, false)
	got := m.makeDeltaArgs(true, 80, deltaOpts{})
	for _, arg := range []string{"--side-by-side", "--syntax-theme=", "--plus-style="} {
		if !slices.ContainsFunc(got, func(a string) bool { return strings.HasPrefix(a, arg) }) {
			t.Errorf("themed: expected %q in %q", arg, got)
		}
	}
}

func TestMakeDeltaEnv(t *testing.T) {
	t.Setenv("GIT_CONFIG_PARAMETERS", "'delta.navigate'='true'")

	common.UseUserDeltaTheme = false
	if got := makeDeltaEnv(true); !slices.Equal(got, os.Environ()) {
		t.Fatalf("themed: expected the plain environment, got %q", got)
	}

	common.UseUserDeltaTheme = true
	t.Cleanup(func() { common.UseUserDeltaTheme = false })
	env := makeDeltaEnv(false)
	got := env[len(env)-1]
	want := "GIT_CONFIG_PARAMETERS='delta.navigate'='true' 'delta.side-by-side'='false'"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestDiffContentMsg_IgnoresStaleResult(t *testing.T) {
	m := newTestModel(t, false)
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
