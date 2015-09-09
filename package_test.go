package gb

import (
	"path/filepath"
	"testing"
)

func testProject(t *testing.T) *Project {
	cwd := getwd(t)
	root := filepath.Join(cwd, "testdata")
	return NewProject(root,
		SourceDir(filepath.Join(root, "src")),
	)
}

func testContext(t *testing.T) *Context {
	prj := testProject(t)
	ctx, err := prj.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	ctx.Force = true
	ctx.SkipInstall = true
	return ctx
}

func TestResolvePackage(t *testing.T) {
	ctx := testContext(t)
	defer ctx.Destroy()
	_, err := ctx.ResolvePackage("a")
	if err != nil {
		t.Fatal(err)
	}
}

func TestPackageName(t *testing.T) {
	ctx := testContext(t)
	defer ctx.Destroy()
	pkg, err := ctx.ResolvePackage("aprime")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := "a", pkg.Name; got != want {
		t.Fatalf("Package.Name(): got %v, want %v", got, want)
	}
}

func TestPackageBinfile(t *testing.T) {
	var tests = []struct {
		goos, goarch string // simulated GOOS and GOARCH values, "" == unset in environment
		pkg          string // package name
		want         string // binfile result
	}{
		{pkg: "b", want: "b"},
	}

	for _, tt := range tests {
		ctx := testContext(t)
		defer ctx.Destroy()
		pkg, err := ctx.ResolvePackage(tt.pkg)
		if err != nil {
			t.Fatal(err)
		}
		got := pkg.Binfile()
		want := filepath.Join(ctx.Bindir(), tt.want)
		if want != got {
			t.Errorf("(%s).Binfile(): want %s, got %s", tt.pkg, want, got)
		}
	}
}
