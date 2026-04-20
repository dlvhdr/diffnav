package common

import (
	"strings"
	"testing"
)

func Test_Scrollbar(t *testing.T) {
	tests := []struct {
		name         string
		want         string
		trackHeight  int
		totalItems   int
		firstItemIdx int
		lastItemIdx  int
	}{
		{
			name:         "No scrollbar as total lines <= height",
			want:         "",
			trackHeight:  3,
			totalItems:   3,
			firstItemIdx: 0,
			lastItemIdx:  2,
		},
		{
			name: "View height is close to the total lines",
			want: `
				┃
				┃
				│
			`,
			trackHeight:  3,
			totalItems:   4,
			firstItemIdx: 0,
			lastItemIdx:  2,
		},
		{
			name: "y offset is at the bottom",
			want: `
				│
				┃
				┃
			`,
			trackHeight:  3,
			totalItems:   4,
			firstItemIdx: 4,
			lastItemIdx:  3,
		},
		{
			name: "y offset is at the bottom with large total lines",
			want: `
				│
				│
				│
				│
				┃
			`,
			trackHeight:  5,
			totalItems:   1000,
			firstItemIdx: 995,
			lastItemIdx:  999,
		},
		{
			name: "y offset is almost at the top with large total lines",
			want: `
				│
				┃
				│
				│
				│
			`,
			trackHeight:  5,
			totalItems:   1000,
			firstItemIdx: 1,
			lastItemIdx:  5,
		},
		{
			name: "y offset is at the middle and track allows bigger thumb size",
			want: `
				│
				│
				│
				┃
				┃
				│
				│
				│
				│
			`,
			trackHeight:  9,
			totalItems:   13,
			firstItemIdx: 2,
			lastItemIdx:  11,
		},
		{
			name: "y offset is almost at the bottom with room to scroll",
			want: `
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				┃
				│
			`,
			trackHeight:  40,
			totalItems:   238,
			firstItemIdx: 197,
			lastItemIdx:  236,
		},
		{
			name: "y offset is at the bottom with large scrollbar without room to scroll",
			want: `
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				│
				┃
			`,
			trackHeight:  40,
			totalItems:   238,
			firstItemIdx: 198,
			lastItemIdx:  237,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb := Scrollbar{}
			want := ""
			lines := strings.Split(tt.want, "\n")
			for i, line := range lines {
				if i == 0 || i == len(lines)-1 {
					continue
				}
				want = want + strings.TrimSpace(line)
				if len(lines) < 2 || i < len(lines)-2 {
					want = want + "\n"
				}

			}
			got := sb.View(tt.trackHeight, tt.totalItems, tt.firstItemIdx, tt.lastItemIdx)
			if want != got {
				t.Fatalf(
					"expected scrollbar to be\n%s\n, but got \n%s\n",
					want,
					got,
				)
			}
		})
	}
}
