package utils

import "testing"

func TestRemoveReset(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "default reset",
			input: "\x1b[31mred\x1b[m",
			want:  "\x1b[31mred",
		},
		{
			name:  "explicit zero reset",
			input: "\x1b[31mred\x1b[0m",
			want:  "\x1b[31mred",
		},
		{
			name:  "multiple reset styles",
			input: "one\x1b[0mtwo\x1b[m",
			want:  "onetwo",
		},
		{
			name:  "no reset",
			input: "plain text",
			want:  "plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RemoveReset(tt.input); got != tt.want {
				t.Fatalf("RemoveReset(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
