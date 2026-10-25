package diffviewer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/dlvhdr/diffnav/pkg/ui/common"
)

func newTestModel(t *testing.T, useDeltaTheme bool) Model {
	t.Helper()
	common.RegisterSupportedTints()
	common.UseDeltaTheme = useDeltaTheme
	t.Cleanup(func() { common.UseDeltaTheme = false })
	styles := common.MakeStyles()
	return New(false, &styles)
}

func hasArgPrefix(args []string, prefix string) bool {
	return slices.ContainsFunc(args, func(a string) bool { return strings.HasPrefix(a, prefix) })
}

func TestMakeDeltaArgs_DeltaThemeOnlyPassesLayoutFlags(t *testing.T) {
	m := newTestModel(t, true)

	want := []string{"--paging=never", "-w=80", "--wrap-max-lines=unlimited"}
	for _, sideBySide := range []bool{false, true} {
		got := m.makeDeltaArgs(sideBySide, 80, deltaOpts{})
		if !slices.Equal(got, want) {
			t.Fatalf("sideBySide=%t: expected %q, got %q", sideBySide, want, got)
		}
	}
}

func TestMakeDeltaArgs_ThemedPassesSideBySideFlag(t *testing.T) {
	m := newTestModel(t, false)

	if got := m.makeDeltaArgs(true, 80, deltaOpts{}); !slices.Contains(got, "--side-by-side") {
		t.Fatalf("expected --side-by-side in %q", got)
	}
}

// unsetEnv removes key for the rest of the test and restores it afterwards.
func unsetEnv(t *testing.T, key string) {
	t.Helper()
	t.Setenv(key, "")
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
}

// lastEnv returns the value exec.Cmd uses for key: the last one in env.
func lastEnv(env []string, key string) (string, bool) {
	for _, kv := range slices.Backward(env) {
		if v, ok := strings.CutPrefix(kv, key+"="); ok {
			return v, true
		}
	}
	return "", false
}

func TestMakeDeltaEnv_DeltaThemeSetsSideBySide(t *testing.T) {
	common.UseDeltaTheme = true
	t.Cleanup(func() { common.UseDeltaTheme = false })
	t.Setenv("GIT_CONFIG_PARAMETERS", "")

	for _, sideBySide := range []bool{false, true} {
		got, _ := lastEnv(makeDeltaEnv(sideBySide), "GIT_CONFIG_PARAMETERS")
		want := fmt.Sprintf("'delta.side-by-side'='%t'", sideBySide)
		if got != want {
			t.Fatalf("expected GIT_CONFIG_PARAMETERS=%q, got %q", want, got)
		}
	}
}

func TestMakeDeltaEnv_DeltaThemeKeepsExistingParameters(t *testing.T) {
	common.UseDeltaTheme = true
	t.Cleanup(func() { common.UseDeltaTheme = false })
	t.Setenv("GIT_CONFIG_PARAMETERS", "'delta.navigate'='true'")

	got, _ := lastEnv(makeDeltaEnv(true), "GIT_CONFIG_PARAMETERS")
	want := "'delta.navigate'='true' 'delta.side-by-side'='true'"
	if got != want {
		t.Fatalf("expected GIT_CONFIG_PARAMETERS=%q, got %q", want, got)
	}
}

func TestMakeDeltaEnv_ThemedIsUnchanged(t *testing.T) {
	common.UseDeltaTheme = false
	t.Setenv("GIT_CONFIG_PARAMETERS", "'delta.navigate'='true'")

	if got := makeDeltaEnv(true); !slices.Equal(got, os.Environ()) {
		t.Fatalf("expected the plain environment, got %q", got)
	}
}

// Runs the real delta to check that side-by-side keeps the user's line-number
// format and can be turned off when the user's config turns it on.
func TestDelta_DeltaThemeSideBySideKeepsUserLineNumbers(t *testing.T) {
	if _, err := exec.LookPath("delta"); err != nil {
		t.Skip("delta not installed")
	}
	m := newTestModel(t, true)
	// Keep the developer's own delta and git settings out of the run.
	t.Setenv("GIT_CONFIG_PARAMETERS", "")
	unsetEnv(t, "GIT_CONFIG_COUNT")
	// An empty DELTA_FEATURES would clear the config's features, so unset it instead.
	unsetEnv(t, "DELTA_FEATURES")
	dir := t.TempDir()

	config := filepath.Join(t.TempDir(), "gitconfig")
	err := os.WriteFile(config, []byte(`[delta]
	features = mine
	side-by-side = true
[delta "mine"]
	line-numbers = true
	line-numbers-left-format = "{nm:>1}L"
	line-numbers-right-format = "{np:>1}R"
`), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	diff := "diff --git a/f b/f\n--- a/f\n+++ b/f\n@@ -1 +1 @@\n-foo bar\n+foo baz\n"
	run := func(sideBySide bool) string {
		args := append(
			[]string{"--config=" + config},
			m.makeDeltaArgs(sideBySide, 80, deltaOpts{})...)
		cmd := exec.Command("delta", args...)
		cmd.Env = makeDeltaEnv(sideBySide)
		cmd.Dir = dir
		cmd.Stdin = strings.NewReader(diff)
		out, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		return ansi.Strip(string(out))
	}

	// Side-by-side: both panels on one line, each with the user's format, not delta's "│ 1  │".
	if out := run(true); !regexp.MustCompile(`1Lfoo bar +1Rfoo baz`).MatchString(out) {
		t.Errorf("expected side-by-side with the user's line numbers, got:\n%s", out)
	}
	// Unified: the user's side-by-side = true is turned off.
	out := run(false)
	if !strings.Contains(out, "1L Rfoo bar") || !strings.Contains(out, " L1Rfoo baz") {
		t.Errorf("expected unified output, got:\n%s", out)
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

func TestDiffContentMsg_IgnoresStaleResults(t *testing.T) {
	m := newTestModel(t, false)
	m.SetSize(80, 10)
	current := &cachedNode{path: "current.go"}
	other := &cachedNode{path: "other.go"}
	m.file = current
	m.cache[cacheKey(current.path, false)] = current
	m.cache[cacheKey(other.path, false)] = other

	m, _ = m.Update(diffContentMsg{cacheKey: "gone.go", lines: stringToDiffLines("stale output")})
	m, _ = m.Update(
		diffContentMsg{
			cacheKey: cacheKey(other.path, false),
			node:     other,
			lines:    stringToDiffLines("other output"),
		},
	)
	if view := m.View(); strings.Contains(view, "stale output") ||
		strings.Contains(view, "other output") {
		t.Fatalf("expected stale and non-current results to stay off screen, got:\n%s", view)
	}
	if len(other.diff) == 0 {
		t.Fatal("expected the non-current result to still be cached")
	}

	m, _ = m.Update(
		diffContentMsg{
			cacheKey: cacheKey(current.path, false),
			node:     current,
			lines:    stringToDiffLines("current output"),
		},
	)
	if view := m.View(); !strings.Contains(view, "current output") {
		t.Fatalf("expected current result on screen, got:\n%s", view)
	}
}

func TestDiffContentMsg_IgnoresOlderRunForSameKey(t *testing.T) {
	m := newTestModel(t, false)
	m.SetSize(80, 10)
	key := cacheKey("a.go", false)
	older := &cachedNode{path: "a.go"}
	newer := &cachedNode{path: "a.go"}
	m.file = newer
	m.cache[key] = newer

	m, _ = m.Update(
		diffContentMsg{cacheKey: key, node: newer, lines: stringToDiffLines("new output")},
	)
	m, _ = m.Update(
		diffContentMsg{cacheKey: key, node: older, lines: stringToDiffLines("old output")},
	)

	if view := m.View(); !strings.Contains(view, "new output") ||
		strings.Contains(view, "old output") {
		t.Fatalf("expected only the newer run on screen, got:\n%s", view)
	}
	if len(older.diff) != 0 {
		t.Fatal("expected the older run not to be cached")
	}
}
