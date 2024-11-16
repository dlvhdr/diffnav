package utils

import (
	"strings"

	"github.com/muesli/reflow/truncate"
)

// TruncateString is a convenient wrapper around truncate.TruncateString.
func TruncateString(s string, max int) string {
	if max < 0 {
		max = 0
	}
	return truncate.StringWithTail(s, uint(max), "…")
}

func RemoveReset(s string) string {
	// Remove ANSI reset codes
	return strings.ReplaceAll(s, "\x1b[m", "")
}
