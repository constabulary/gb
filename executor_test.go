package gb

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"
)

func TestExecuteBuildAction(t *testing.T) {
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
		action, err := BuildPackages(pkg)
		if err != nil {
			t.Errorf("BuildAction(%v): ", tt.pkg, err)
			continue
		}
		if err := Execute(action); !sameErr(err, tt.err) {
			t.Errorf("Execute(%v): want: %v, got %v", action.Name, tt.err, err)
		}
		ctx.Destroy()
	}
}
