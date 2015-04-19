package gb

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestTestPackage(t *testing.T) {
	tests := []struct {
		pkg string
		err error
	}{{
		pkg: "a",
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
		pkg := ctx.ResolvePackage(tt.pkg)
		err := Test(pkg).Result()
		if err != tt.err {
			t.Errorf("Test(tt.pkg): want %v, got %v", tt.err, err)
		}
	}
}
