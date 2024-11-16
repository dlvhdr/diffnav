package filetreev2

import (
	"os"
	"testing"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/dlvhdr/diffnav/pkg/constants"
	"github.com/dlvhdr/diffnav/pkg/filenode"
)

// .
// ├── graphql-server
// │   └── tests
// │       └── package.json
// ├── yarn.lock
func TestBuildFullFileTree(t *testing.T) {
	f, err := os.Open("testdata/multiple_files.diff")
	if err != nil {
		t.Fatal(err)
	}
	files, _, err := gitdiff.Parse(f)
	if err != nil {
		t.Fatal(err)
	}

	tr := buildFullFileTree(files, options{})
	allNodes := tr.AllNodes()
	if len(allNodes) != 5 {
		t.Fatalf("expected 5 nodes, but got %d", len(allNodes))
	}
	root := tr
	if root.GivenValue() != constants.RootName {
		t.Fatalf(`expected root value to be constants.RootName, but got "%s"`, root.Value())
	}

	if len(root.ChildNodes()) != 2 {
		t.Fatalf("expected root to have 2 children, but got %d", len(root.ChildNodes()))
	}

	graphqlServer := root.ChildNodes()[0]
	if graphqlServer.GivenValue() != "graphql-server" {
		t.Fatalf(`expected root first child value to be "graphql-server", but got %s`, graphqlServer.GivenValue())
	}
	yarnLock := root.ChildNodes()[1]
	if yarnLock.GivenValue().(*filenode.FileNode).Path() != "yarn.lock" {
		t.Log(tr.String())
		t.Fatalf(`expected root second child value to be "* yarn.lock", but got %s`,
			yarnLock.GivenValue().(*filenode.FileNode).Path())
	}

	if len(graphqlServer.ChildNodes()) != 1 {
		t.Fatalf("expected graphql-server to have 1 children, but got %d", len(graphqlServer.ChildNodes()))
	}

	tests := graphqlServer.ChildNodes()[0]
	if tests.GivenValue() != "tests" {
		t.Fatalf(`expected graphql-server only child value to be "tests", but got %s`, tests.GivenValue())
	}

	if len(tests.ChildNodes()) != 1 {
		t.Fatalf("expected tests to have 1 children, but got %d", len(tests.ChildNodes()))
	}

	packageJson := tests.ChildNodes()[0]
	if packageJson.GivenValue().(*filenode.FileNode).Path() != "graphql-server/tests/package.json" {
		t.Fatalf(`expected tests only child value to be "graphql-server/tests/package.json", but got %s`,
			packageJson.GivenValue().(*filenode.FileNode).Path())
	}
}

// input:
// .
// ├── graphql-server
// │   └── tests
// │       └── package.json
// └── yarn.lock
//
// output:
// .
// ├── graphql-server/tests
// │   └── package.json
// └── yarn.lock
func TestCollapseTree(t *testing.T) {
	f, err := os.Open("testdata/multiple_files.diff")
	if err != nil {
		t.Fatal(err)
	}
	files, _, err := gitdiff.Parse(f)
	if err != nil {
		t.Fatal(err)
	}

	tr := buildFullFileTree(files, options{})
	tr = collapseTree(tr)

	allNodes := tr.AllNodes()
	if len(allNodes) != 4 {
		t.Fatalf("expected 4 nodes, but got %d", len(allNodes))
	}

	root := tr
	if root.GivenValue() != constants.RootName {
		t.Fatalf(`expected root value to be constants.RootName, but got "%s"`, root.Value())
	}

	if len(root.ChildNodes()) != 2 {
		t.Fatalf("expected root to have 2 children, but got %d", len(root.ChildNodes()))
	}

	graphqlServer := root.ChildNodes()[0]
	if graphqlServer.GivenValue() != "graphql-server/tests" {
		t.Fatalf(`expected root first child value to be "graphql-server/tests", but got %s`, graphqlServer.GivenValue())
	}

	if len(graphqlServer.ChildNodes()) != 1 {
		t.Fatalf("expected graphql-server to have 1 children, but got %d", len(graphqlServer.ChildNodes()))
	}
	packageJson := graphqlServer.ChildNodes()[0]
	if packageJson.GivenValue().(*filenode.FileNode).Path() != "graphql-server/tests/package.json" {
		t.Fatalf(`expected graphql-server/tests only child value to be "graphql-server/tests/package.json", but got %s`, packageJson.GivenValue())
	}

	yarnLock := root.ChildNodes()[1]
	if yarnLock.GivenValue().(*filenode.FileNode).Path() != "yarn.lock" {
		t.Log(tr.String())
		t.Fatalf(`expected root second child value to be "* yarn.lock", but got %s`,
			yarnLock.GivenValue().(*filenode.FileNode).Path())
	}
}

// input:
// .
// └── ui
//     ├── components
//     │   ├── reposection
//     │   │   ├── commands.go
//     │   │   └── reposection.go
//     │   ├── section
//     │   │   └── section.go
//     │   └── tasks
//     │       └── pr.go
//     └─ keys
//     │   └── branchkeys.go
//     └── ui.go

// output is the same as there are no collapsible nodes
func TestUncollapsableTree(t *testing.T) {
	f, err := os.Open("testdata/gh_dash_pr.diff")
	if err != nil {
		t.Fatal(err)
	}
	files, _, err := gitdiff.Parse(f)
	if err != nil {
		t.Fatal(err)
	}

	tr := buildFullFileTree(files, options{})

	tr = collapseTree(tr)
	allNodes := tr.AllNodes()
	if len(allNodes) != 13 {
		t.Fatalf("expected 13 nodes, but got %d", len(allNodes))
	}
}
