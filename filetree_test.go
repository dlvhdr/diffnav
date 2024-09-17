package main

import (
	"testing"

	"github.com/charmbracelet/lipgloss/tree"
)

func TestEmptyTree(t *testing.T) {
	files := []string{}

	want := `.`
	got := buildFullFileTree(files).String()

	if got != want {
		t.Errorf("files:\n%v\n\n------- want:\n%v\n\n-------got:\n%v\n", files, want, got)
	}
}

func TestBuildTreeSingleFile(t *testing.T) {
	files := []string{
		"main.go",
	}

	want := `.
└── main.go`
	got := buildFullFileTree(files).String()

	if got != want {
		t.Errorf("files:\n%v\n\n------- want:\n%v\n\n-------got:\n%v\n", files, want, got)
	}
}

func TestFileWithinDir(t *testing.T) {
	files := []string{"ui/main.go"}

	want := tree.Root("ui").
		Child("main.go")
	got := buildFullFileTree(files)
	got = collapseTree(got)
	compareTree(t, want, got)
}

func TestFileWithinNestedDirAndRootFile(t *testing.T) {
	files := []string{"ui/components/main.go", "cmd.go"}

	want := tree.Root(".").
		Child("cmd.go").
		Child(
			tree.Root("ui/components").
				Child("main.go"))

	got := buildFullFileTree(files)
	got = collapseTree(got)
	compareTree(t, want, got)
}

func TestFileWithMultipleFiles(t *testing.T) {
	files := []string{"main.go", "cmd.go"}

	want := tree.Root(".").
		Child("cmd.go").
		Child("main.go")

	got := buildFullFileTree(files)
	got = collapseTree(got)
	compareTree(t, want, got)
}

func TestFileWithNestedDirWithFile(t *testing.T) {
	files := []string{"components/main.go", "components/subdir/comp.go"}

	want := tree.Root(".").
		Child(tree.Root("components").
			Child("main.go").
			Child(tree.Root("subdir").
				Child("comp.go")))
	got := buildFullFileTree(files).String()

	if got != want.String() {
		t.Errorf("files:\n%v\n\n------- want:\n%v\n\n-------got:\n%v\n", files, want, got)
	}
}

func TestDeeplyNestedFile(t *testing.T) {
	files := []string{"components/main.go", "components/subdir/comp.go", "components/subdir/subsubdir/deepcomp.go"}

	want := tree.Root(".").
		Child(tree.Root("components").
			Child("main.go").
			Child(tree.Root("subdir").
				Child("comp.go").
				Child(tree.Root("subsubdir").
					Child("deepcomp.go"))))
	got := buildFullFileTree(files).String()

	if got != want.String() {
		t.Errorf("files:\n%v\n\n------- want:\n%v\n\n-------got:\n%v\n", files, want, got)
	}
}

func TestComplex(t *testing.T) {
	files := []string{
		"ui/components/a.go",
		"ui/components/b.go",
		"ui/components/sub/c.go",
		"ui/main.go",
		"utils/misc/pointers.go",
		"utils/misc/sorters.go",
		"pkg/internal/ws.go",
	}

	want := tree.Root(".").
		Child(tree.Root("pkg/internal").
			Child("ws.go")).
		Child(tree.Root("ui").
			Child("main.go").
			Child(tree.Root("components").
				Child("a.go").
				Child("b.go").
				Child(tree.Root("sub").
					Child("c.go")))).
		Child(tree.Root("utils/misc").
			Child("pointers.go").
			Child("sorters.go"))
	got := buildFullFileTree(files)
	got = collapseTree(got)
	compareTree(t, want, got)
}

func TestDirectChild(t *testing.T) {
	files := []string{"main.go"}
	want := tree.Root(".").Child("main.go").String()
	got := buildFullFileTree(files).String()

	if got != want {
		t.Errorf("want:\n%v\n\n-------got:\n%v\n", want, got)
	}
}

func TestChildWithMultiDirectory(t *testing.T) {
	files := []string{"ui/components/subdir/comp.go", "ui/main.go"}
	want := tree.Root("ui").
		Child("main.go").
		Child(tree.Root("components/subdir").
			Child("comp.go"))
	got := buildFullFileTree(files)
	got = collapseTree(got)
	compareTree(t, want, got)
}

func TestCommonAncestor(t *testing.T) {
	files := []string{
		"ui/components/subdir/section.go",
		"ui/components/subdir/pr.go",
		"ui/components/tasks/task/task.go",
	}
	want := tree.Root("ui/components").
		Child(tree.Root("subdir").
			Child("pr.go").
			Child("section.go")).
		Child(tree.Root("tasks/task").
			Child("task.go"))

	got := buildFullFileTree(files)
	got = collapseTree(got)
	compareTree(t, want, got)
}

func TestCommonAncestorSorting(t *testing.T) {
	files := []string{
		"ui/comp/subdir/pr.go",
		"ui/z/section.go",
	}
	want := tree.Root(".").
		Child(tree.Root("ui").
			Child(tree.Root("comp").
				Child(tree.Root("subdir").Child("pr.go"))).
			Child(tree.Root("z").
				Child("section.go")),
		).
		String()

	got := buildFullFileTree(files).String()
	if got != want {
		t.Errorf("want:\n%v\n\n-------got:\n%v\n", want, got)
	}
}

func TestGhData(t *testing.T) {
	files := []string{
		"ui/components/reposection/commands.go",
		"ui/components/reposection/reposection.go",
		"ui/components/section/section.go",
		"ui/components/tasks/pr.go",
		"ui/keys/branchKeys.go",
	}
	want := tree.Root(".").
		Child(tree.Root("ui").
			Child(tree.Root("components").
				Child(tree.Root("reposection").
					Child("commands.go").
					Child("reposection.go")).
				Child(tree.Root("section").
					Child("section.go")).
				Child(tree.Root("tasks").
					Child("pr.go"))).
			Child(tree.Root("keys").
				Child("branchKeys.go"))).String()

	got := buildFullFileTree(files).String()
	if got != want {
		t.Errorf("want:\n%v\n\n-------got:\n%v\n", want, got)
	}
}

func TestPrintWithoutRoot(t *testing.T) {
	files := []string{
		"ui/components/a.go",
		"ui/components/b.go",
		"ui/components/sub/c.go",
		"ui/main.go",
		"utils/misc/pointers.go",
		"utils/misc/sorters.go",
		"pkg/internal/ws.go",
	}

	want := `pkg/internal
└── ws.go
ui
├── main.go
└── components
    ├── a.go
    ├── b.go
    └── sub
        └── c.go
utils/misc
├── pointers.go
└── sorters.go`

	got := buildFullFileTree(files)
	got = collapseTree(got)
	printed := printWithoutRoot(got)

	if printed != want {
		t.Errorf("want:\n%v\n\n-------got:\n%v\n", want, got)
	}
}

func TestPrintWithoutRootKeepsCommonRoot(t *testing.T) {
	files := []string{
		"ui/components/a.go",
		"ui/components/b.go",
		"ui/components/sub/c.go",
		"ui/main.go",
	}

	want := `ui
├── main.go
└── components
    ├── a.go
    ├── b.go
    └── sub
        └── c.go`

	got := buildFullFileTree(files)
	got = collapseTree(got)

	printed := printWithoutRoot(got)

	if printed != want {
		t.Errorf("want:\n%v\n-------got:\n%v\n", want, printed)
	}
}

func TestGetDirStructureOneFile(t *testing.T) {
	files := []string{
		"ui/main.go",
	}
	want := tree.Root(".").
		Child(tree.Root("ui").
			Child("main.go")).String()

	got := buildFullFileTree(files).String()
	if got != want {
		t.Errorf("want:\n%v\n\n-------got:\n%v\n", want, got)
	}
}

func TestGetDirStructureTwoUnrelatedFiles(t *testing.T) {
	files := []string{
		"ui/main.go",
		"pkg/cmd.go",
	}
	want := tree.Root(".").
		Child(tree.Root("pkg").
			Child("cmd.go")).
		Child(tree.Root("ui").
			Child("main.go")).String()

	got := buildFullFileTree(files).String()
	if got != want {
		t.Errorf("want:\n%v\n\n-------got:\n%v\n", want, got)
	}
}

func TestBuildFullComplexTree(t *testing.T) {
	files := []string{
		"ui/components/reposection/commands.go",
		"ui/components/reposection/reposection.go",
		"ui/components/section/section.go",
		"ui/components/tasks/pr.go",
		"ui/keys/branchKeys.go",
	}
	want := tree.Root(".").
		Child(tree.Root("ui").
			Child(tree.Root("components").
				Child(tree.Root("reposection").
					Child("commands.go").
					Child("reposection.go")).
				Child(tree.Root("section").
					Child("section.go")).
				Child(tree.Root("tasks").
					Child("pr.go"))).
			Child(tree.Root("keys").
				Child("branchKeys.go"))).String()

	got := buildFullFileTree(files).String()

	if got != want {
		t.Errorf("want:\n%v\n\n-------got:\n%v\n", want, got)
	}
}

func TestCollapseUncollapsibleTree(t *testing.T) {
	input := tree.Root(".").
		Child(tree.Root("pkg").
			Child("cmd.go")).
		Child(tree.Root("ui").
			Child("main.go"))
	want := tree.Root(".").
		Child(tree.Root("pkg").
			Child("cmd.go")).
		Child(tree.Root("ui").
			Child("main.go")).String()

	collapseTree(input)
	if input.String() != want {
		t.Errorf("want:\n%v\n\n-------got:\n%v\n", want, input)
	}
}

func TestCollapsibleComplexTree(t *testing.T) {
	input := tree.Root(".").
		Child(tree.Root("ui").
			Child(tree.Root("components").
				Child(tree.Root("reposection").
					Child("commands.go")).
				Child(tree.Root("tasks").
					Child("pr.go"))))

	want := tree.Root("ui/components").
		Child(tree.Root("reposection").
			Child("commands.go")).
		Child(tree.Root("tasks").
			Child("pr.go"))

	got := collapseTree(input)
	if got.String() != want.String() {
		t.Errorf("want:\n%v\n\n-------got:\n%v\n", want, got)
	}
}

func TestCollapsibleTree(t *testing.T) {
	input := tree.Root(".").
		Child(tree.Root("ui").
			Child(tree.Root("subdir").
				Child("pr.go").
				Child("section.go")))
	want := tree.Root("ui/subdir").
		Child("pr.go").
		Child("section.go")

	got := collapseTree(input)
	if got.String() != want.String() {
		t.Errorf("want:\n%v\n\n-------got:\n%v\n", want, got)
	}
}

func compareTree(t *testing.T, want, got tree.Node) {
	if got.String() != want.String() {
		t.Errorf("want:\n%v\n\n-------got:\n%v\n", want, got)
	}
}
