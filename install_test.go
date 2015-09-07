package gb

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStale(t *testing.T) {
	var tests = []struct {
		pkgs  []string
		stale map[string]bool
	}{{
		pkgs: []string{"a"},
		stale: map[string]bool{
			"a": true,
		},
	}, {
		pkgs: []string{"a", "b"},
		stale: map[string]bool{
			"a": false,
			"b": true,
		},
	}, {
		pkgs: []string{"a", "b"},
		stale: map[string]bool{
			"a": false,
			"b": false,
		},
	}}

	root := mktemp(t)
	defer os.RemoveAll(root)

	proj := Project{
		rootdir: root,
		srcdirs: []Srcdir{{
			Root: filepath.Join(getwd(t), "testdata", "src"),
		}},
	}

	newctx := func() *Context {
		ctx, err := proj.NewContext(
			GcToolchain(),
		)
		if err != nil {
			t.Fatal(err)
		}
		return ctx
	}

	resolve := func(ctx *Context, pkg string) *Package {
		p, err := ctx.ResolvePackage(pkg)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}

	for _, tt := range tests {
		ctx := newctx()
		defer ctx.Destroy()
		for _, pkg := range tt.pkgs {
			resolve(ctx, pkg)
		}

		for p, s := range tt.stale {
			pkg := resolve(ctx, p)
			if pkg.Stale != s {
				t.Errorf("%q.Stale: got %v, want %v", pkg.Name, pkg.Stale, s)
			}
		}

		for _, pkg := range tt.pkgs {
			if err := Build(resolve(ctx, pkg)); err != nil {
				t.Fatal(err)
			}
		}
	}
}
