package gb

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/pkg/errors"
)

func testContext(t *testing.T, opts ...func(*Context) error) *Context {
	ctx, err := NewContext(testProject(t), opts...)
	if err != nil {
		t.Fatal(err)
	}
	ctx.Force = true
	return ctx
}

func TestResolvePackage(t *testing.T) {
	var tests = []struct {
		pkg  string // package name
		opts []func(*Context) error
		err  error
	}{{
		pkg: "a",
	}, {
		pkg: "localimport",
		err: &importErr{path: "../localimport", msg: "relative import not supported"},
	}}
	proj := testProject(t)
	for _, tt := range tests {
		ctx, err := NewContext(proj, tt.opts...)
		defer ctx.Destroy()
		_, err = ctx.ResolvePackage(tt.pkg)
		err = errors.Cause(err)
		if !reflect.DeepEqual(err, tt.err) {
			t.Errorf("ResolvePackage(%q): want: %v, got %v", tt.pkg, tt.err, err)
		}
	}
}

func TestPackageBinfile(t *testing.T) {
	opts := func(o ...func(*Context) error) []func(*Context) error { return o }
	gotargetos := "windows"
	if runtime.GOOS == "windows" {
		gotargetos = "linux"
	}
	gotargetarch := "386"
	if runtime.GOARCH == "386" {
		gotargetarch = "amd64"
	}
	var tests = []struct {
		pkg  string // package name
		opts []func(*Context) error
		want string // binfile result
	}{{
		pkg:  "b",
		want: "b",
	}, {
		pkg:  "b",
		opts: opts(GOOS(gotargetos)),
		want: fmt.Sprintf("b-%v-%v", gotargetos, runtime.GOARCH),
	}, {
		pkg:  "b",
		opts: opts(GOARCH(gotargetarch)),
		want: fmt.Sprintf("b-%v-%v", runtime.GOOS, gotargetarch),
	}, {
		pkg:  "b",
		opts: opts(GOARCH(gotargetarch), GOOS(gotargetos)),
		want: fmt.Sprintf("b-%v-%v", gotargetos, gotargetarch),
	}, {
		pkg:  "b",
		opts: opts(Tags("lol")),
		want: "b-lol",
	}, {
		pkg:  "b",
		opts: opts(GOARCH(gotargetarch), GOOS(gotargetos), Tags("lol")),
		want: fmt.Sprintf("b-%v-%v-lol", gotargetos, gotargetarch),
	}}

	proj := testProject(t)
	for i, tt := range tests {
		ctx, _ := NewContext(proj, tt.opts...)
		defer ctx.Destroy()
		pkg, err := ctx.ResolvePackage(tt.pkg)
		if err != nil {
			t.Fatal(err)
		}
		got := pkg.Binfile()
		want := filepath.Join(ctx.bindir(), tt.want)
		if pkg.gotargetos == "windows" {
			want += ".exe"
		}
		if want != got {
			t.Errorf("test %v: (%s).Binfile(): want %s, got %s", i+1, tt.pkg, want, got)
		}
	}
}

func TestPackageBindir(t *testing.T) {
	ctx := testContext(t)
	defer ctx.Destroy()
	tests := []struct {
		pkg  *Package
		want string
	}{{
		pkg: &Package{
			Context: ctx,
		},
		want: ctx.bindir(),
	}, {
		pkg: &Package{
			Package: &build.Package{
				Name:       "testpkg",
				ImportPath: "github.com/constabulary/gb/testpkg",
			},
			Context:   ctx,
			TestScope: true,
		},
		want: filepath.Join(ctx.Workdir(), "github.com", "constabulary", "gb", "testpkg", "_test"),
	}}

	for i, tt := range tests {
		got := tt.pkg.bindir()
		want := tt.want
		if got != want {
			t.Errorf("test %v: Bindir: got %v want %v", i+1, got, want)
		}
	}
}

func TestNewPackage(t *testing.T) {
	tests := []struct {
		pkg  build.Package
		want Package
	}{{
		pkg: build.Package{
			Name:       "C",
			ImportPath: "C",
			Goroot:     true,
		},
		want: Package{
			NotStale: true,
		},
	}}
	proj := testProject(t)
	for i, tt := range tests {
		ctx, _ := NewContext(proj)
		defer ctx.Destroy()

		got, err := ctx.NewPackage(&tt.pkg)
		if err != nil {
			t.Error(err)
			continue
		}
		want := tt.want // deep copy
		want.Package = &tt.pkg
		want.Context = ctx

		if !reflect.DeepEqual(got, &want) {
			t.Errorf("%d: pkg: %s: expected %#v, got %#v", i+1, tt.pkg.ImportPath, &want, got)
		}
	}
}

func TestStale(t *testing.T) {
	var tests = []struct {
		pkgs  []string
		stale map[string]bool
	}{{
		pkgs: []string{"a"},
		stale: map[string]bool{
			"a": false,
		},
	}, {
		pkgs: []string{"a", "b"},
		stale: map[string]bool{
			"a": true,
			"b": false,
		},
	}, {
		pkgs: []string{"a", "b"},
		stale: map[string]bool{
			"a": true,
			"b": true,
		},
	}}

	proj := tempProject(t)
	defer os.RemoveAll(proj.rootdir)
	proj.tempfile("src/a/a.go", `package a

const A = "A"
`)

	proj.tempfile("src/b/b.go", `package main

import "a"

func main() {
        println(a.A)
}
`)

	newctx := func() *Context {
		ctx, err := NewContext(proj,
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
		ctx.Install = true
		defer ctx.Destroy()
		for _, pkg := range tt.pkgs {
			resolve(ctx, pkg)
		}

		for p, s := range tt.stale {
			pkg := resolve(ctx, p)
			if pkg.NotStale != s {
				t.Errorf("%q.NotStale: got %v, want %v", pkg.Name, pkg.NotStale, s)
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
		got := pkg.installpath()
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
		got := pkg.pkgpath()
		want := tt.pkgpath(ctx)
		if got != want {
			t.Errorf("pkgpath(%q): expected: %v, got %v", tt.pkg, want, got)
		}
	}
}

func TestPackageIncludePaths(t *testing.T) {
	ctx := testContext(t)
	tests := []struct {
		pkg  *Package
		want []string
	}{{
		pkg: &Package{
			Context: ctx,
			Package: &build.Package{
				ImportPath: "github.com/foo/bar",
			},
		},
		want: []string{
			ctx.Workdir(),
			ctx.Pkgdir(),
		},
	}, {
		pkg: &Package{
			Context: ctx,
			Package: &build.Package{
				ImportPath: "github.com/foo/bar",
			},
			Main: true,
		},
		want: []string{
			ctx.Workdir(),
			ctx.Pkgdir(),
		},
	}, {
		pkg: &Package{
			Context: ctx,
			Package: &build.Package{
				ImportPath: "github.com/foo/bar",
			},
			TestScope: true,
		},
		want: []string{
			filepath.Join(ctx.Workdir(), "github.com", "foo", "bar", "_test"),
			ctx.Workdir(),
			ctx.Pkgdir(),
		},
	}, {
		pkg: &Package{
			Context: ctx,
			Package: &build.Package{
				ImportPath: "github.com/foo/bar",
			},
			TestScope: true,
			Main:      true,
		},
		want: []string{
			filepath.Join(ctx.Workdir(), "github.com", "foo", "_test"), // TODO(dfc) WTF
			ctx.Workdir(),
			ctx.Pkgdir(),
		},
	}}

	for i, tt := range tests {
		got := tt.pkg.includePaths()
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%d: Package: ImportPath: %v, TestScope: %v, Main: %v: got %v, want %v", i, tt.pkg.ImportPath, tt.pkg.TestScope, tt.pkg.Main, got, tt.want)
		}
	}
}
