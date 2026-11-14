package diffviewer

import (
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

func osc(body string) string { return osc1717Prefix + body + osc1717Term }

func TestParseDeltaOutput(t *testing.T) {
	out := osc1717Handshake + osc("1;f;;;a.go") + "a.go\n" +
		osc("1;h;10;;a.go") + "@@\n" +
		"\x1b[0m────\n" +
		osc("1;d;10;9;a.go") + "\x1b[31m-x" + osc("1;a;10;;a.go") + "+y\n" +
		osc("1;c;11;;dir;b.go") + " z\n" +
		osc("1;C;;;abc123") + osc("2;c;1;;a.go") + "commit"

	lines := parseDeltaOutput(out)
	if len(lines) != 6 {
		t.Fatalf("expected 6 rows, got %d", len(lines))
	}
	want := [][]record{
		{{Kind: 'f', File: "a.go"}},
		{{Kind: 'h', NewLine: 10, File: "a.go"}},
		nil,
		{{Kind: 'd', NewLine: 10, OldLine: 9, File: "a.go"}, {Kind: 'a', NewLine: 10, File: "a.go"}},
		{{Kind: 'c', NewLine: 11, File: "dir;b.go"}},
		nil, // unknown type and version
	}
	for i, l := range lines {
		if !reflect.DeepEqual(l.records, want[i]) {
			t.Errorf("row %d: expected %+v, got %+v", i, want[i], l.records)
		}
	}
}

func TestParseRecords_StripsText(t *testing.T) {
	text, _ := parseRecords(osc("1;d;10;9;a.go") + "\x1b[31m-x" + osc("1;a;10;;a.go") + "+y")
	if text != "\x1b[31m-x+y" {
		t.Fatalf("expected records stripped, got %q", text)
	}
}

func TestParseDeltaOutput_NoHandshake(t *testing.T) {
	lines := parseDeltaOutput("plain\n" + osc("1;c;1;;a.go") + " x")
	if len(lines) != 2 || lines[1].records != nil {
		t.Fatalf("expected no records without the handshake, got %+v", lines)
	}
}

func TestParseDeltaOutput_RealDelta(t *testing.T) {
	if _, err := exec.LookPath("delta"); err != nil {
		t.Skip("delta not installed")
	}
	t.Setenv("GIT_CONFIG_PARAMETERS", "")
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	patch := "diff --git a/a;b.go b/a;b.go\n--- a/a;b.go\n+++ b/a;b.go\n" +
		"@@ -1,2 +1,2 @@\n ctx\n-old\n+new\n"
	cmd := exec.Command("delta", "--paging=never", "--side-by-side", "-w=80")
	cmd.Env = makeDeltaEnv(deltaOpts{})
	cmd.Stdin = strings.NewReader(patch)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("delta: %v", err)
	}
	if !strings.HasPrefix(string(out), osc1717Handshake) {
		t.Skip("delta has no OSC 1717 support")
	}

	var got []record
	for _, l := range parseDeltaOutput(string(out)) {
		for _, r := range l.records {
			if r.Kind == 'c' || r.Kind == 'a' || r.Kind == 'd' {
				got = append(got, r)
			}
		}
	}
	want := []record{
		{Kind: 'c', NewLine: 1, File: "a;b.go"},
		{Kind: 'c', NewLine: 1, File: "a;b.go"},
		{Kind: 'd', NewLine: 2, OldLine: 2, File: "a;b.go"},
		{Kind: 'a', NewLine: 2, File: "a;b.go"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %+v, got %+v", want, got)
	}
}
