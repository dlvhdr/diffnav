package ui

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"

	"github.com/dlvhdr/diffnav/pkg/filenode"
)

func sortFiles(files []*gitdiff.File) {
	slices.SortFunc(files, func(a *gitdiff.File, b *gitdiff.File) int {
		nameA := filenode.GetFileName(a)
		nameB := filenode.GetFileName(b)
		dira := filepath.Dir(nameA)
		dirb := filepath.Dir(nameB)
		if dira != "." && dirb != "." && dira == dirb {
			return strings.Compare(strings.ToLower(nameA), strings.ToLower(nameB))
		}

		if dira != "." && dirb == "." {
			return -1
		}
		if dirb != "." && dira == "." {
			return 1
		}

		if dira != "." && dirb != "." {
			if strings.HasPrefix(dira, dirb) {
				return -1
			}

			if strings.HasPrefix(dirb, dira) {
				return 1
			}
		}

		return strings.Compare(strings.ToLower(nameA), strings.ToLower(nameB))
	})
}
