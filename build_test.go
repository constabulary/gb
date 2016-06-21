package gb

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/constabulary/gb/internal/importer"
)

func TestBuild(t *testing.T) {
	opts := func(o ...func(*Context) error) []func(*Context) error { return o }
	tests := []struct {
		pkg  string
		opts []func(*Context) error
		err  error
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
		pkg: "notestfiles",
		err: nil,
	}, {
		pkg: "cgoonlynotest",
		err: nil,
	}, {
		pkg: "testonly",
		err: nil,
	}, {
		pkg: "extestonly",
		err: nil,
	}, {
		pkg: "mainnoruntime",
		err: nil,
	}, {
		pkg: "h", // imports "blank", which is blank, see issue #131
		err: &importer.NoGoError{filepath.Join(getwd(t), "testdata", "src", "blank")},
	}, {
		pkg: "cppmain",
	}, {
		pkg:  "tags1",
		opts: opts(Tags("x")), // excludes the test file in package
		err:  &importer.NoGoError{filepath.Join(getwd(t), "testdata", "src", "tags1")},
	}, {
		pkg: "tags2",
		err: &importer.NoGoError{filepath.Join(getwd(t), "testdata", "src", "tags2")},
	}, {
		pkg:  "tags2",
		opts: opts(Tags("x")),
	}, {
		pkg: "nosource",
		err: &importer.NoGoError{filepath.Join(getwd(t), "testdata", "src", "nosource")},
	}}

	proj := testProject(t)
	for _, tt := range tests {
		ctx, err := NewContext(proj, tt.opts...)
		ctx.Force = true
		defer ctx.Destroy()
		pkg, err := ctx.ResolvePackage(tt.pkg)
		if !reflect.DeepEqual(err, tt.err) {
			t.Errorf("ctx.ResolvePackage(%v): want %v, got %v", tt.pkg, tt.err, err)
			continue
		}
		if err != nil {
			continue
		}
		if err := Build(pkg); !reflect.DeepEqual(err, tt.err) {
			t.Errorf("ctx.Build(%v): want %v, got %v", tt.pkg, tt.err, err)
		}
	}
}

func TestBuildPackage(t *testing.T) {
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
		pkg: "cgomain",
		err: nil,
	}, {
		pkg: "cgotest",
		err: nil,
	}, {
		pkg: "notestfiles",
		err: nil,
	}, {
		pkg: "cgoonlynotest",
		err: nil,
	}, {
		pkg: "testonly",
		err: errors.New(`compile "testonly": no go files supplied`),
	}, {
		pkg: "extestonly",
		err: errors.New(`compile "extestonly": no go files supplied`),
	}}

	for _, tt := range tests {
		ctx := testContext(t)
		defer ctx.Destroy()
		pkg, err := ctx.ResolvePackage(tt.pkg)
		if err != nil {
			t.Errorf("ctx.ResolvePackage(%v):  %v", tt.pkg, err)
			continue
		}
		targets := make(map[string]*Action)
		if _, err := BuildPackage(targets, pkg); !reflect.DeepEqual(err, tt.err) {
			t.Errorf("ctx.BuildPackage(%v): want %v, got %v", tt.pkg, tt.err, err)
		}
	}
}

func TestBuildPackages(t *testing.T) {
	tests := []struct {
		pkgs    []string
		actions []string
		options []func(*Context) error // set of options to apply to the test context
		err     error
	}{{
		pkgs:    []string{"a", "b", "c"},
		actions: []string{"compile: a", "compile: c", "link: b"},
	}, {
		pkgs:    []string{"cgotest", "cgomain", "notestfiles", "cgoonlynotest", "testonly", "extestonly"},
		actions: []string{"compile: notestfiles", "link: cgomain", "pack: cgoonlynotest", "pack: cgotest"},
	}, {
		pkgs:    []string{"a", "b", "c"},
		options: []func(*Context) error{WithRace},
		actions: []string{"compile: a", "compile: c", "link: b"},
	}}

	for _, tt := range tests {
		ctx := testContext(t, tt.options...)
		defer ctx.Destroy()
		var pkgs []*Package
		for _, pkg := range tt.pkgs {
			pkg, err := ctx.ResolvePackage(pkg)
			if err != nil {
				t.Errorf("ctx.ResolvePackage(%v):  %v", pkg, err)
				continue
			}
			pkgs = append(pkgs, pkg)
		}
		a, err := BuildPackages(pkgs...)
		if !reflect.DeepEqual(err, tt.err) {
			t.Errorf("ctx.BuildPackages(%v): want %v, got %v", pkgs, tt.err, err)
		}
		var names []string
		for _, a := range a.Deps {
			names = append(names, a.Name)
		}
		sort.Strings(names)
		if !reflect.DeepEqual(tt.actions, names) {
			t.Errorf("ctx.BuildPackages(%v): want %v, got %v", pkgs, tt.actions, names)
		}
	}
}

func TestObjfile(t *testing.T) {
	var tests = []struct {
		pkg  string // package name
		want string // objfile result
	}{
		{pkg: "b", want: "b/main.a"},
		{pkg: "nested/a", want: "nested/a.a"},
		{pkg: "nested/b", want: "nested/b.a"},
	}

	for _, tt := range tests {
		ctx := testContext(t)
		defer ctx.Destroy()
		pkg, err := ctx.ResolvePackage(tt.pkg)
		if err != nil {
			t.Fatal(err)
		}
		got := objfile(pkg)
		want := filepath.Join(ctx.Workdir(), tt.want)
		if want != got {
			t.Errorf("(%s).Objdir(): want %s, got %s", tt.pkg, want, got)
		}
	}
}

func TestCgoobjdir(t *testing.T) {
	var tests = []struct {
		pkg  string // package name
		want string // objdir result
	}{
		{pkg: "b", want: "b/_cgo"},
		{pkg: "nested/a", want: "nested/a/_cgo"},
		{pkg: "nested/b", want: "nested/b/_cgo"},
	}

	ctx := testContext(t)
	defer ctx.Destroy()
	for _, tt := range tests {
		pkg, err := ctx.ResolvePackage(tt.pkg)
		if err != nil {
			t.Fatal(err)
		}
		got := cgoworkdir(pkg)
		want := filepath.Join(ctx.Workdir(), tt.want)
		if want != got {
			t.Errorf("(%s).cgoobjdir(): want %s, got %s", tt.pkg, want, got)
		}
	}
}

func TestWorkdir(t *testing.T) {
	var tests = []struct {
		pkg  string // package name
		want string // objdir result
	}{
		{pkg: "b", want: ""},
		{pkg: "nested/a", want: "nested"},
		{pkg: "nested/b", want: "nested"},
	}

	ctx := testContext(t)
	defer ctx.Destroy()
	for _, tt := range tests {
		pkg, err := ctx.ResolvePackage(tt.pkg)
		if err != nil {
			t.Error(err)
			continue
		}
		got := Workdir(pkg)
		want := filepath.Join(ctx.Workdir(), tt.want)
		if want != got {
			t.Errorf("Workdir(Package{Name: %v, ImportPath: %v, TestScope: %v}): want %s, got %s", pkg.Name, pkg.ImportPath, pkg.TestScope, want, got)
		}
	}
}

func TestPkgname(t *testing.T) {
	var tests = []struct {
		pkg  *Package
		want string
	}{{
		pkg: &Package{
			Package: &importer.Package{
				Name:       "main",
				ImportPath: "main",
			},
		},
		want: "main",
	}, {
		pkg: &Package{
			Package: &importer.Package{
				Name:       "a",
				ImportPath: "main",
			},
		},
		want: "a",
	}, {
		pkg: &Package{
			Package: &importer.Package{
				Name:       "main",
				ImportPath: "a",
			},
		},
		want: "a",
	}, {
		pkg: &Package{
			Package: &importer.Package{
				Name:       "main",
				ImportPath: "testmain",
			},
		},
		want: "testmain",
	}, {
		pkg: &Package{
			Package: &importer.Package{
				Name:       "main",
				ImportPath: "main",
			},
			TestScope: true,
		},
		want: "main",
	}, {
		pkg: &Package{
			Package: &importer.Package{
				Name:       "a",
				ImportPath: "main",
			},
			TestScope: true,
		},
		want: "main",
	}, {
		pkg: &Package{
			Package: &importer.Package{
				Name:       "main",
				ImportPath: "a",
			},
			TestScope: true,
		},
		want: "a",
	}, {
		pkg: &Package{
			Package: &importer.Package{
				Name:       "main",
				ImportPath: "a/a",
			},
			TestScope: true,
		},
		want: "a",
	}, {
		pkg: &Package{
			Package: &importer.Package{
				Name:       "main",
				ImportPath: "testmain",
			},
			TestScope: true,
		},
		want: "testmain",
	}}

	for _, tt := range tests {
		got := pkgname(tt.pkg)
		if got != tt.want {
			t.Errorf("pkgname(Package{Name:%q, ImportPath: %q, TestScope:%v}): got %v, want %v", tt.pkg.Name, tt.pkg.ImportPath, tt.pkg.TestScope, got, tt.want)
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

func mktemp(t *testing.T) string {
	s, err := mktmp()
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func mktmp() (string, error) {
	return ioutil.TempDir("", "gb-test-")
}
