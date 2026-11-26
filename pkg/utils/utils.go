package utils

import (
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/truncate"
)

var ansiResetReplacer = strings.NewReplacer(
	"\x1b[0m", "",
	"\x1b[m", "",
)

// TruncateString is a convenient wrapper around truncate.TruncateString.
func TruncateString(s string, max int) string {
	if runewidth.StringWidth(s) <= max {
		return s
	}
	if max < 0 {
		max = 0
	}
	return truncate.StringWithTail(s, uint(max), "…")
}

func RemoveReset(s string) string {
	return ansiResetReplacer.Replace(s)
}
