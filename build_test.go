package gb

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestBuild(t *testing.T) {
	tests := []struct {
		pkg string
		err error
	}{{
		pkg: "a",
		err: nil,
	}, {
		pkg: "b", // actually command
		err: nil,
	}, {
		pkg: "c",
		err: nil,
	}}

	root, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	proj := NewProject(root)

	tc, err := NewGcToolchain(runtime.GOROOT(), runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range tests {
		ctx := proj.NewContext(tc)
		ctx.Force = true
		ctx.SkipInstall = true
		pkg, err := ctx.ResolvePackage(tt.pkg)
		if err != tt.err {
			t.Errorf("ctx.ResolvePackage(tt.pkg): want %v, got %v", tt.err, err)
			continue
		}
		if err := Build(pkg); err != tt.err {
			t.Errorf("ctx.Build(tt.pkg): want %v, got %v", tt.err, err)
		}
	}
}
