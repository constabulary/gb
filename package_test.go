package gb

import (
	"os"
	"path/filepath"
	"testing"
)

func testProject(t *testing.T) *Project {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(cwd, "testdata")
	return &Project{
		rootdir: root,
	}
}

func testContext(t *testing.T) *Context {
	prj := testProject(t)
	ctx, err := prj.NewContext(
		GcToolchain(),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx.Force = true
	ctx.SkipInstall = true
	return ctx
}

func TestResolvePackage(t *testing.T) {
	ctx := testContext(t)
	_, err := ctx.ResolvePackage("a")
	if err != nil {
		t.Fatal(err)
	}
}

func TestPackageName(t *testing.T) {
	ctx := testContext(t)
	pkg, err := ctx.ResolvePackage("aprime")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := "a", pkg.Name; got != want {
		t.Fatalf("Package.Name(): got %v, want %v", got, want)
	}
}
