package diffviewer

import (
	"slices"
	"strconv"
	"strings"

	"github.com/robinovitch61/viewport/viewport/item"
)

// OSC 1717 diff line metadata (https://github.com/stefanhaller/diff-line-metadata-spec).
// Delta >= 0.20 emits it when OSC1717=V1 is set; older versions ignore the variable.
const (
	osc1717Prefix    = "\x1b]1717;"
	osc1717Term      = "\x1b\\"
	osc1717Handshake = osc1717Prefix + "1" + osc1717Term
)

// record says which diff line a row shows.
// Kind is c (context), a (added), d (deleted), f (file header) or h (hunk header).
type record struct {
	Kind    byte
	NewLine int
	OldLine int
	File    string
}

// parseDeltaOutput splits delta's output into rows. When it starts with the
// OSC 1717 handshake (a prefix of the first row, not a row of its own), each
// row's records are parsed and stripped from its text.
func parseDeltaOutput(out string) []diffLine {
	rest, ok := strings.CutPrefix(out, osc1717Handshake)
	if !ok {
		return stringToDiffLines(out)
	}
	rows := strings.Split(rest, "\n")
	lines := make([]diffLine, len(rows))
	for i, row := range rows {
		text, records := parseRecords(row)
		lines[i] = diffLine{item: item.NewItem(text), records: records}
	}
	return lines
}

// parseRecords removes the OSC 1717 sequences from a row and returns them.
func parseRecords(row string) (string, []record) {
	if !strings.Contains(row, osc1717Prefix) {
		return row, nil
	}
	var records []record
	var b strings.Builder
	for {
		start := strings.Index(row, osc1717Prefix)
		if start < 0 {
			break
		}
		end := strings.Index(row[start:], osc1717Term)
		if end < 0 {
			break
		}
		b.WriteString(row[:start])
		if r, ok := parseRecord(row[start+len(osc1717Prefix) : start+end]); ok {
			records = append(records, r)
		}
		row = row[start+end+len(osc1717Term):]
	}
	b.WriteString(row)
	return b.String(), records
}

// parseRecord parses "<version>;<type>;<new>;<old>;<file>".
func parseRecord(body string) (record, bool) {
	fields := strings.SplitN(body, ";", 5)
	if len(fields) != 5 || fields[0] != "1" || len(fields[1]) != 1 ||
		!strings.Contains("cadfh", fields[1]) {
		return record{}, false
	}
	newLine, _ := strconv.Atoi(fields[2])
	oldLine, _ := strconv.Atoi(fields[3])
	file := fields[4]
	// Git quotes unusual names with octal escapes (\303\244 for ä), and delta copies them,
	// with the quotes for a renamed file.
	if strings.Contains(file, `\`) {
		quoted := file
		if len(file) < 2 || file[0] != '"' || file[len(file)-1] != '"' {
			quoted = `"` + file + `"`
		}
		if name, err := strconv.Unquote(quoted); err == nil {
			file = name
		}
	}
	return record{Kind: fields[1][0], NewLine: newLine, OldLine: oldLine, File: file}, true
}

// firstRecord returns the first record at or below row.
func firstRecord(lines []diffLine, row int) (record, bool) {
	for i := max(row, 0); i < len(lines); i++ {
		if len(lines[i].records) > 0 {
			return lines[i].records[0], true
		}
	}
	return record{}, false
}

// findRecord returns the first row showing the same diff line as want, or -1.
func findRecord(lines []diffLine, want record) int {
	for i, l := range lines {
		if l.shows(want) {
			return i
		}
	}
	return -1
}

// shows reports whether the row shows the same diff line as want.
func (o diffLine) shows(want record) bool {
	for _, r := range o.records {
		if r.File == want.File && sameLine(r, want) {
			return true
		}
	}
	return false
}

// sameLine compares records of the same file: d by old line, c/a and h by new line.
func sameLine(r, want record) bool {
	switch want.Kind {
	case 'd':
		return r.Kind == 'd' && r.OldLine == want.OldLine
	case 'c', 'a':
		return (r.Kind == 'c' || r.Kind == 'a') && r.NewLine == want.NewLine
	default:
		return r.Kind == want.Kind && r.NewLine == want.NewLine
	}
}

// hunkRow returns the first row of the next hunk after row (dir > 0) or of the previous
// one before it (dir < 0), or -1.
func hunkRow(lines []diffLine, row, dir int) int {
	starts := hunkStarts(lines)
	if dir < 0 {
		slices.Reverse(starts)
	}
	for _, s := range starts {
		if (s-row)*dir > 0 {
			return s
		}
	}
	return -1
}

// hunkStarts returns the first row of each hunk: the first row of its header (delta
// decorates it over several rows with the same h record) or, when delta printed no
// header, its first line. Delta skips the header of a hunk with no code fragment
// when the hunk-header style shows no line number or file, as diffnav's own does.
// Without a header, a hunk starts after a file header, where the new line jumps, or
// where a deleted line skips an old line. An addition-only hunk right before a
// deletion hunk (possible with -U0) is still missed: added lines have no old line.
func hunkStarts(lines []diffLine) []int {
	var starts []int
	var hunk record      // the current hunk's h record, if it has one
	file, last := "", -1 // the current hunk's file and highest new line, -1 after a header
	old := -1            // the next old line, -1 until a deleted line gives it
	open := false
	for i, l := range lines {
		if len(l.records) == 0 {
			continue
		}
		if h, ok := hunkRecord(l); ok {
			if !open || h != hunk {
				starts = append(starts, i)
			}
			open, hunk, file, last, old = true, h, h.File, -1, -1
			continue
		}
		r := l.records[0]
		if r.Kind == 'f' {
			open = false
			continue
		}
		if i > 0 && slices.Equal(l.records, lines[i-1].records) {
			continue // a wrapped row of the same line
		}
		n, del := 0, record{}
		for _, r := range l.records {
			n = max(n, r.NewLine)
			if r.Kind == 'd' {
				del = r
			}
		}
		// Within a hunk, new lines go up by at most one per line and deleted lines
		// follow the old lines.
		if !open || r.File != file || (last >= 0 && n > last+1) ||
			(del.Kind == 'd' && old >= 0 && del.OldLine > old) {
			starts = append(starts, i)
			open, hunk, file, last, old = true, record{}, r.File, n, -1
		}
		last = max(last, n)
		switch {
		case del.Kind == 'd':
			old = del.OldLine + 1
		case r.Kind == 'c' && old >= 0:
			old++
		}
	}
	return starts
}

// hunkRecord returns the row's h record, if any.
func hunkRecord(l diffLine) (record, bool) {
	for _, r := range l.records {
		if r.Kind == 'h' {
			return r, true
		}
	}
	return record{}, false
}
