package gb

import (
	"os"
	"path/filepath"
	"runtime"
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

func TestInstallpath(t *testing.T) {
	ctx := testContext(t)
	defer ctx.Destroy()

	tests := []struct {
		pkg         string
		installpath string
	}{{
		pkg:         "a", // from testdata
		installpath: filepath.Join(ctx.Pkgdir(), "a.a"),
	}, {
		pkg:         "runtime", // from stdlib
		installpath: filepath.Join(ctx.Pkgdir(), "runtime.a"),
	}, {
		pkg:         "unsafe", // synthetic
		installpath: filepath.Join(ctx.Pkgdir(), "unsafe.a"),
	}}

	resolve := func(pkg string) *Package {
		p, err := ctx.ResolvePackage(pkg)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}

	for _, tt := range tests {
		pkg := resolve(tt.pkg)
		got := installpath(pkg)
		if got != tt.installpath {
			t.Errorf("installpath(%q): expected: %v, got %v", tt.pkg, tt.installpath, got)
		}
	}
}

func TestPkgpath(t *testing.T) {
	opts := func(o ...func(*Context) error) []func(*Context) error { return o }
	gotargetos := "windows"
	if runtime.GOOS == gotargetos {
		gotargetos = "linux"
	}
	gotargetarch := "arm64"
	if runtime.GOARCH == "arm64" {
		gotargetarch = "amd64"
	}
	tests := []struct {
		opts    []func(*Context) error
		pkg     string
		pkgpath func(*Context) string
	}{{
		pkg:     "a", // from testdata
		pkgpath: func(ctx *Context) string { return filepath.Join(ctx.Pkgdir(), "a.a") },
	}, {
		opts:    opts(GOOS(gotargetos), GOARCH(gotargetarch)),
		pkg:     "a", // from testdata
		pkgpath: func(ctx *Context) string { return filepath.Join(ctx.Pkgdir(), "a.a") },
	}, {
		opts:    opts(WithRace),
		pkg:     "a", // from testdata
		pkgpath: func(ctx *Context) string { return filepath.Join(ctx.Pkgdir(), "a.a") },
	}, {
		opts:    opts(Tags("foo", "bar")),
		pkg:     "a", // from testdata
		pkgpath: func(ctx *Context) string { return filepath.Join(ctx.Pkgdir(), "a.a") },
	}, {
		pkg: "runtime", // from stdlib
		pkgpath: func(ctx *Context) string {
			return filepath.Join(runtime.GOROOT(), "pkg", ctx.gohostos+"_"+ctx.gohostarch, "runtime.a")
		},
	}, {
		opts: opts(Tags("foo", "bar")),
		pkg:  "runtime", // from stdlib
		pkgpath: func(ctx *Context) string {
			return filepath.Join(runtime.GOROOT(), "pkg", ctx.gohostos+"_"+ctx.gohostarch, "runtime.a")
		},
	}, {
		opts: opts(WithRace),
		pkg:  "runtime", // from stdlib
		pkgpath: func(ctx *Context) string {
			return filepath.Join(runtime.GOROOT(), "pkg", ctx.gohostos+"_"+ctx.gohostarch+"_race", "runtime.a")
		},
	}, {
		opts: opts(WithRace, Tags("foo", "bar")),
		pkg:  "runtime", // from stdlib
		pkgpath: func(ctx *Context) string {
			return filepath.Join(runtime.GOROOT(), "pkg", ctx.gohostos+"_"+ctx.gohostarch+"_race", "runtime.a")
		},
	}, {
		opts: opts(GOOS(gotargetos), GOARCH(gotargetarch)),
		pkg:  "runtime", // from stdlib
		pkgpath: func(ctx *Context) string {
			return filepath.Join(ctx.Pkgdir(), "runtime.a")
		},
	}, {
		pkg: "unsafe", // synthetic
		pkgpath: func(ctx *Context) string {
			return filepath.Join(runtime.GOROOT(), "pkg", ctx.gohostos+"_"+ctx.gohostarch, "unsafe.a")
		},
	}, {
		pkg:  "unsafe", // synthetic
		opts: opts(GOOS(gotargetos), GOARCH(gotargetarch), WithRace),
		pkgpath: func(ctx *Context) string {
			return filepath.Join(ctx.Pkgdir(), "unsafe.a")
		},
	}}

	for _, tt := range tests {
		ctx := testContext(t, tt.opts...)
		defer ctx.Destroy()
		pkg, err := ctx.ResolvePackage(tt.pkg)
		if err != nil {
			t.Fatal(err)
		}
		got := pkgpath(pkg)
		want := tt.pkgpath(ctx)
		if got != want {
			t.Errorf("pkgpath(%q): expected: %v, got %v", tt.pkg, want, got)
		}
	}
}
