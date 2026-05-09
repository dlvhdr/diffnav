package cmd

import "testing"

func TestIsUnifiedDiff(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name: "git diff (unified)",
			input: "diff --git a/foo.go b/foo.go\n" +
				"index 1234567..89abcde 100644\n" +
				"--- a/foo.go\n" +
				"+++ b/foo.go\n" +
				"@@ -1,3 +1,4 @@\n" +
				" package foo\n" +
				"+\n" +
				" func A() {}\n",
			want: true,
		},
		{
			name: "git show (preamble + unified)",
			input: "commit abc1234567890\n" +
				"Author: Someone <s@example.com>\n" +
				"Date:   Mon Jan 1 00:00:00 2026 +0000\n" +
				"\n" +
				"    subject\n" +
				"\n" +
				"diff --git a/foo.go b/foo.go\n" +
				"@@ -1 +1,2 @@\n" +
				" a\n+b\n",
			want: true,
		},
		{
			name: "git diff --stat",
			input: " main.go | 5 ++---\n" +
				" foo.go  | 1 +\n" +
				" 2 files changed, 3 insertions(+), 3 deletions(-)\n",
			want: false,
		},
		{
			name:  "git diff --shortstat",
			input: " 2 files changed, 3 insertions(+), 3 deletions(-)\n",
			want:  false,
		},
		{
			name:  "git diff --name-only",
			input: "main.go\nfoo.go\n",
			want:  false,
		},
		{
			name:  "git diff --name-status",
			input: "M\tmain.go\nA\tfoo.go\n",
			want:  false,
		},
		{
			name: "git log (no patch)",
			input: "commit abc1234567890\n" +
				"Author: Someone <s@example.com>\n" +
				"Date:   Mon Jan 1 00:00:00 2026 +0000\n" +
				"\n" +
				"    subject\n",
			want: false,
		},
		{
			name:  "empty",
			input: "",
			want:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isUnifiedDiff(tc.input)
			if got != tc.want {
				t.Fatalf("isUnifiedDiff(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}
