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

// lastEnv returns the value exec.Cmd uses for key: the last one in env.
func lastEnv(env []string, key string) (string, bool) {
	for _, kv := range slices.Backward(env) {
		if v, ok := strings.CutPrefix(kv, key+"="); ok {
			return v, true
		}
	}
	return "", false
}

func TestMakeDeltaEnv_OwnConfigSetsSideBySide(t *testing.T) {
	common.DeltaOwnConfig = true
	t.Cleanup(func() { common.DeltaOwnConfig = false })
	t.Setenv("GIT_CONFIG_PARAMETERS", "")

	for _, sideBySide := range []bool{false, true} {
		got, _ := lastEnv(makeDeltaEnv(sideBySide), "GIT_CONFIG_PARAMETERS")
		want := fmt.Sprintf("'delta.side-by-side'='%t'", sideBySide)
		if got != want {
			t.Fatalf("expected GIT_CONFIG_PARAMETERS=%q, got %q", want, got)
		}
	}
}

func TestMakeDeltaEnv_OwnConfigKeepsExistingParameters(t *testing.T) {
	common.DeltaOwnConfig = true
	t.Cleanup(func() { common.DeltaOwnConfig = false })
	t.Setenv("GIT_CONFIG_PARAMETERS", "'delta.navigate'='true'")

	got, _ := lastEnv(makeDeltaEnv(true), "GIT_CONFIG_PARAMETERS")
	want := "'delta.navigate'='true' 'delta.side-by-side'='true'"
	if got != want {
		t.Fatalf("expected GIT_CONFIG_PARAMETERS=%q, got %q", want, got)
	}
}

func TestMakeDeltaEnv_ThemedIsUnchanged(t *testing.T) {
	common.DeltaOwnConfig = false
	t.Setenv("GIT_CONFIG_PARAMETERS", "'delta.navigate'='true'")

	if got := makeDeltaEnv(true); !slices.Equal(got, os.Environ()) {
		t.Fatalf("expected the plain environment, got %q", got)
	}
}

// Runs the real delta to check that side-by-side keeps the user's line-number
// format and can be turned off when the user's config turns it on.
func TestDelta_OwnConfigSideBySideKeepsUserLineNumbers(t *testing.T) {
	if _, err := exec.LookPath("delta"); err != nil {
		t.Skip("delta not installed")
	}
	m := newTestModel(t, true)
	t.Setenv("GIT_CONFIG_PARAMETERS", "")

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
	m, _ = m.Update(
		diffContentMsg{
			cacheKey: cacheKey(other.path, false),
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
			lines:    stringToDiffLines("current output"),
		},
	)
	if view := m.View(); !strings.Contains(view, "current output") {
		t.Fatalf("expected current result on screen, got:\n%s", view)
	}
}
