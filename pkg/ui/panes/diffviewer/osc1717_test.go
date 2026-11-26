package diffviewer

import (
	"os/exec"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/robinovitch61/viewport/viewport/item"
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
		{
			{Kind: 'd', NewLine: 10, OldLine: 9, File: "a.go"},
			{Kind: 'a', NewLine: 10, File: "a.go"},
		},
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

func TestParseRecords_UnquotesFileName(t *testing.T) {
	for raw, want := range map[string]string{
		`\303\244.txt`:   "ä.txt",
		`"\303\244.txt"`: "ä.txt",   // a rename keeps git's quotes
		`a\q.txt`:        `a\q.txt`, // not a valid escape: kept as is
	} {
		_, records := parseRecords(osc("1;c;1;;"+raw) + "x")
		if len(records) != 1 || records[0].File != want {
			t.Errorf("%s: expected file %q, got %+v", raw, want, records)
		}
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
	patch := "diff --git a/a;b.go b/a;b.go\n--- a/a;b.go\n+++ b/a;b.go\n" +
		"@@ -1,2 +1,2 @@\n ctx\n-old\n+new\n"
	t.Setenv("GIT_CONFIG_PARAMETERS", "")
	cmd := exec.Command("delta", "--no-gitconfig", "--paging=never", "--side-by-side", "-w=80")
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

func row(records ...record) diffLine {
	return diffLine{item: item.NewItem("x"), records: records}
}

// ctxLine is a context line n of a.go.
func ctxLine(n int) record { return record{Kind: 'c', NewLine: n, File: "a.go"} }

// unifiedAndSbs returns lines 1 to 20 of a.go in unified, and side by side with a
// decoration row before every line.
func unifiedAndSbs() (unified, sbs []diffLine) {
	for n := 1; n <= 20; n++ {
		unified = append(unified, row(ctxLine(n)))
		sbs = append(sbs, row(), row(ctxLine(n), ctxLine(n)))
	}
	return unified, sbs
}

// showing returns a model of the given height that shows lines as a.go in unified.
func showing(t *testing.T, height int, lines []diffLine) Model {
	m := newTestModel(t)
	m.SetSize(80, height)
	m.file = &cachedNode{path: "a.go", files: []*gitdiff.File{{}}, diff: lines}
	m.cache[cacheKey("a.go", false)] = m.file
	m.setLines(lines)
	return m
}

func TestFindRecord(t *testing.T) {
	a := func(n int) record { return record{Kind: 'a', NewLine: n, File: "a.go"} }
	d := func(n, o int) record { return record{Kind: 'd', NewLine: n, OldLine: o, File: "a.go"} }
	// Side-by-side: a modification is one row with d and a.
	lines := []diffLine{
		row(record{Kind: 'f', File: "a.go"}),
		row(record{Kind: 'h', NewLine: 1, File: "a.go"}),
		row(ctxLine(1), ctxLine(1)),
		row(d(2, 2), a(2)),
		row(a(3)),
		row(),
		row(ctxLine(4), ctxLine(4)),
	}
	tests := []struct {
		name string
		want record
		row  int
	}{
		{"deleted by old line", d(2, 2), 3},
		{"added on the right", a(3), 4},
		{"hunk", record{Kind: 'h', NewLine: 1, File: "a.go"}, 1},
		{"file", record{Kind: 'f', File: "a.go"}, 0},
		{"past the end", ctxLine(9), -1},
		{"other file", record{Kind: 'c', NewLine: 1, File: "b.go"}, -1},
	}
	for _, tt := range tests {
		if got := findRecord(lines, tt.want); got != tt.row {
			t.Errorf("%s: expected row %d, got %d", tt.name, tt.row, got)
		}
	}
}

func TestSetSideBySide_KeepsPosition(t *testing.T) {
	m := newTestModel(t)
	m.SetSize(80, 4)
	var unified, sbs []diffLine
	for n := 1; n <= 20; n++ {
		unified = append(unified, row(ctxLine(n)))
		if n == 7 {
			unified = append(unified, row(ctxLine(n))) // line 7 wraps in unified only
		}
		sbs = append(sbs, row(), row(ctxLine(n), ctxLine(n))) // a decoration row before every line
	}
	files := []*gitdiff.File{{}}
	m.file = &cachedNode{path: "a.go", files: files}
	m.cache[cacheKey("a.go", false)] = m.file
	m.cache[cacheKey("a.go", true)] = &cachedNode{path: "a.go", files: files, diff: sbs}
	m, _ = m.Update(diffContentMsg{cacheKey: cacheKey("a.go", false), node: m.file, lines: unified})
	m.ScrollDown(7) // line 7's wrapped row at the top

	m.SetSideBySide(true)
	if got := m.fvp.GetItemMetrics().FirstVisibleItemIdx; got != 13 {
		t.Fatalf("expected row 13 (line 7) at the top, got %d", got)
	}
}

func TestSetSideBySide_KeepsRecordlessTopRow(t *testing.T) {
	// A preamble, then line 1 wrapped over two rows.
	lines := []diffLine{row(), row(), row(ctxLine(1)), row(ctxLine(1))}
	for n := 2; n <= 20; n++ {
		lines = append(lines, row(ctxLine(n)))
	}
	for _, top := range []int{0, 3} {
		m := newTestModel(t)
		m.SetSize(80, 4)
		// A new file renders the same in both modes, so `s` re-shows the same lines.
		m.file = &cachedNode{path: "a.go", files: []*gitdiff.File{{IsNew: true}}, diff: lines}
		m.cache[cacheKey("a.go", false)] = m.file
		m.setLines(lines)
		m.ScrollDown(top)

		m.SetSideBySide(true)
		if got := m.fvp.GetItemMetrics().FirstVisibleItemIdx; got != top {
			t.Errorf("expected row %d to stay at the top, got %d", top, got)
		}
	}
}

func TestSetSideBySide_UserMoveDuringRender(t *testing.T) {
	unified, sbs := unifiedAndSbs()
	for _, key := range []string{"g", "G", "j"} {
		m := showing(t, 6, unified)
		m.ScrollDown(10)

		m.SetSideBySide(true) // renders async: the anchor is pending
		m, _ = m.Update(tea.KeyPressMsg{Code: rune(key[0]), Text: key})
		want, _ := m.currentRecord()
		m, _ = m.Update(diffContentMsg{cacheKey: cacheKey("a.go", true), node: m.file, lines: sbs})
		metrics := m.fvp.GetItemMetrics()
		got, _ := firstRecord(sbs, metrics.FirstVisibleItemIdx)
		switch {
		case key == "g" && metrics.FirstVisibleItemIdx != 0:
			t.Errorf("g: expected the top, got row %d", metrics.FirstVisibleItemIdx)
		case key == "G" && metrics.LastVisibleItemIdx != len(sbs)-1:
			t.Errorf("G: expected the bottom, got row %d", metrics.LastVisibleItemIdx)
		case key == "j" && got != want:
			t.Errorf("j: expected line %d at the top, got line %d", want.NewLine, got.NewLine)
		}
	}
}

func TestSetSideBySide_WheelWithoutMoveKeepsAnchor(t *testing.T) {
	unified, sbs := unifiedAndSbs()
	m := showing(t, 4, unified)
	m.fvp.GoToBottom()
	want, _ := firstRecord(unified, m.fvp.GetItemMetrics().FirstVisibleItemIdx)

	m.SetSideBySide(true)
	m.ScrollDown(3) // the wheel at the bottom doesn't move
	m, _ = m.Update(diffContentMsg{cacheKey: cacheKey("a.go", true), node: m.file, lines: sbs})
	if got, _ := firstRecord(sbs, m.fvp.GetItemMetrics().FirstVisibleItemIdx); got != want {
		t.Fatalf("expected line %d at the top, got line %d", want.NewLine, got.NewLine)
	}
}

func TestSwitchToUncachedNode_HasNoRecord(t *testing.T) {
	a := []diffLine{row(record{Kind: 'c', NewLine: 40, File: "a.go"})}
	b := &gitdiff.File{OldName: "b.go", NewName: "b.go"}
	switches := map[string]func(Model) Model{
		"file": func(m Model) Model { m, _ = m.SetFilePatch(b); return m },
		"dir":  func(m Model) Model { m, _ = m.SetDirPatch("pkg", []*gitdiff.File{b}); return m },
	}
	for name, switchTo := range switches {
		m := newTestModel(t)
		m.SetSize(80, 6)
		m.setLines(a)
		m = switchTo(m)
		if rec, ok := m.currentRecord(); ok {
			t.Errorf("%s: expected no record before the render lands, got %+v", name, rec)
		}
	}
}

func TestSetSideBySide_EdgeKeyAtTheEdge(t *testing.T) {
	unified, sbs := unifiedAndSbs()
	for _, key := range []string{"g", "G"} {
		m := showing(t, 4, unified)
		if key == "G" {
			m.fvp.GoToBottom()
		}

		m.SetSideBySide(true)
		m, _ = m.Update(tea.KeyPressMsg{Code: rune(key[0]), Text: key}) // already there
		m, _ = m.Update(diffContentMsg{cacheKey: cacheKey("a.go", true), node: m.file, lines: sbs})
		metrics := m.fvp.GetItemMetrics()
		if key == "g" && metrics.FirstVisibleItemIdx != 0 {
			t.Errorf("g: expected the top, got row %d", metrics.FirstVisibleItemIdx)
		}
		if key == "G" && metrics.LastVisibleItemIdx != len(sbs)-1 {
			t.Errorf("G: expected the bottom, got row %d", metrics.LastVisibleItemIdx)
		}
	}
}
