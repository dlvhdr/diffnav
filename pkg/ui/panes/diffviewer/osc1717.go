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
		before, after, ok := strings.Cut(row, osc1717Prefix)
		if !ok {
			break
		}
		body, rest, ok := strings.Cut(after, osc1717Term)
		if !ok {
			break // unterminated: kept as text
		}
		b.WriteString(before)
		if r, ok := parseRecord(body); ok {
			records = append(records, r)
		}
		row = rest
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
	r := record{Kind: fields[1][0], NewLine: newLine, OldLine: oldLine, File: unquote(fields[4])}
	return r, true
}

// unquote undoes git's quoting of unusual file names (\303\244 for ä), which delta copies,
// with the quotes for a renamed file and without them otherwise. Invalid escapes stay as is.
func unquote(file string) string {
	if !strings.Contains(file, `\`) {
		return file
	}
	quoted := file
	if !strings.HasPrefix(file, `"`) {
		quoted = `"` + file + `"`
	}
	if name, err := strconv.Unquote(quoted); err == nil {
		return name
	}
	return file
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
	return slices.IndexFunc(lines, func(l diffLine) bool { return l.shows(want) })
}

// shows reports whether the row shows the same diff line as want.
func (o diffLine) shows(want record) bool {
	return slices.ContainsFunc(o.records, func(r record) bool { return sameLine(r, want) })
}

// sameLine reports whether r is the same diff line as want: in the same file, d by old line,
// c and a by new line, f and h by kind and new line.
func sameLine(r, want record) bool {
	if r.File != want.File {
		return false
	}
	switch want.Kind {
	case 'd':
		return r.Kind == 'd' && r.OldLine == want.OldLine
	case 'c', 'a':
		return (r.Kind == 'c' || r.Kind == 'a') && r.NewLine == want.NewLine
	default:
		return r.Kind == want.Kind && r.NewLine == want.NewLine
	}
}
