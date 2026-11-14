package diffviewer

import (
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

// hunkRow returns the first row of the next hunk header after row (dir > 0) or the
// previous one before it (dir < 0), or -1. A header can span several rows (delta's
// decorations), all with the same h record.
func hunkRow(lines []diffLine, row, dir int) int {
	for i := row + dir; i >= 0 && i < len(lines); i += dir {
		if h, ok := hunkRecord(lines[i]); ok && (i == 0 || !lines[i-1].shows(h)) {
			return i
		}
	}
	return -1
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
