package gb

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestBuild(t *testing.T) {
	Verbose = false
	defer func() { Verbose = false }()
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
	}, {
		pkg: "d.v1",
		err: nil,
	}, {
		pkg: "x",
		err: errors.New("import cycle detected: x -> y -> x"),
	}, {
		pkg: "cgomain",
		err: nil,
	}, {
		pkg: "cgotest",
		err: nil,
	}, {
		pkg: "h", // imports "blank", which is blank, see issue #131
		err: fmt.Errorf("no buildable Go source files in %s", filepath.Join(getwd(t), "testdata", "src", "blank")),
	}}

	for _, tt := range tests {
		ctx := testContext(t)
		pkg, err := ctx.ResolvePackage(tt.pkg)
		if !sameErr(err, tt.err) {
			t.Errorf("ctx.ResolvePackage(%v): want %v, got %v", tt.pkg, tt.err, err)
			continue
		}
		if err != nil {
			continue
		}
		if err := Build(pkg); !sameErr(err, tt.err) {
			t.Errorf("ctx.Build(%v): want %v, got %v", tt.pkg, tt.err, err)
		}
		ctx.Destroy()
	}
}

func sameErr(e1, e2 error) bool {
	if e1 != nil && e2 != nil {
		return e1.Error() == e2.Error()
	}
	return e1 == e2
}

func getwd(t *testing.T) string {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return cwd
}
