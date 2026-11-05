package ui

import "testing"

func TestMatchesQuery(t *testing.T) {
	tests := []struct {
		name  string
		file  string
		query string
		want  bool
	}{
		{"empty query", "pkg/ui/tui.go", "", true},
		{"only spaces", "pkg/ui/tui.go", "   ", true},
		{"include match", "pkg/ui/tui.go", "tui", true},
		{"include miss", "pkg/ui/tui.go", "keys", false},
		{"exclude match", "pkg/ui/tui_test.go", "!test", false},
		{"exclude miss", "pkg/ui/tui.go", "!test", true},
		{"mixed keeps", "pkg/ui/tui.go", "ui !test", true},
		{"mixed drops excluded", "pkg/ui/tui_test.go", "ui !test", false},
		{"mixed drops missing include", "pkg/config/config.go", "ui !test", false},
		{"case insensitive include", "pkg/ui/TUI.go", "tui", true},
		{"case insensitive exclude", "pkg/ui/tui_Test.go", "!TEST", false},
		{"lone bang ignored", "pkg/ui/tui.go", "!", true},
		{"extra spaces", "pkg/ui/tui.go", "  ui   !test  ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesQuery(tt.file, tt.query); got != tt.want {
				t.Fatalf("matchesQuery(%q, %q) = %v, want %v", tt.file, tt.query, got, tt.want)
			}
		})
	}
}
