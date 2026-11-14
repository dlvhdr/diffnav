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
// OSC 1717 handshake, each row's records are parsed and stripped from its text.
func parseDeltaOutput(out string) []diffLine {
	rest, ok := strings.CutPrefix(out, osc1717Handshake)
	if !ok {
		return stringToDiffLines(out)
	}
	rest = strings.TrimPrefix(rest, "\n")
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
	return record{Kind: fields[1][0], NewLine: newLine, OldLine: oldLine, File: fields[4]}, true
}
