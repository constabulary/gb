package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/constabulary/gb"
)

func TestResolvePackages(t *testing.T) {
	cwd := getwd(t)
	root := filepath.Join(cwd, "..", "testdata", "src")
	tests := []struct {
		paths []string
		err   error
	}{
		{paths: []string{"a"}},
		{paths: []string{"."}, err: fmt.Errorf("%q is not a package", root)},
	}

	for _, tt := range tests {
		ctx := testContext(t)
		_, err := ResolvePackages(ctx, tt.paths...)
		if !sameErr(err, tt.err) {
			t.Errorf("ResolvePackage(%v): want: %v, got %v", tt.paths, tt.err, err)
		}
	}
}

func getwd(t *testing.T) string {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return cwd
}

func testProject(t *testing.T) *gb.Project {
	cwd := getwd(t)
	root := filepath.Join(cwd, "..", "testdata")
	return gb.NewProject(root)
}

func testContext(t *testing.T) *gb.Context {
	prj := testProject(t)
	ctx, err := prj.NewContext(
		gb.GcToolchain(),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx.Force = true
	ctx.SkipInstall = true
	return ctx
}
