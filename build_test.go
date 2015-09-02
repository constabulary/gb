package gb

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
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
		pkg: "h", // imports "blank", which is blank, see issue #131
		err: fmt.Errorf("no buildable Go source files in %s", filepath.Join(getwd(t), "testdata", "src", "blank")),
	}}

	for _, tt := range tests {
		ctx := testContext(t)
		defer ctx.Destroy()
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
	}
}

func TestBuildPackage(t *testing.T) {
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
		if _, err := BuildPackage(targets, pkg); !sameErr(err, tt.err) {
			t.Errorf("ctx.BuildPackage(%v): want %v, got %v", tt.pkg, tt.err, err)
		}
	}
}

func TestBuildPackages(t *testing.T) {
	Verbose = false
	defer func() { Verbose = false }()
	tests := []struct {
		pkgs    []string
		actions []string
		err     error
	}{{
		pkgs:    []string{"a", "b", "c"},
		actions: []string{"compile: a", "compile: c", "link: b"},
	}, {
		pkgs:    []string{"cgotest", "cgomain", "notestfiles", "cgoonlynotest", "testonly", "extestonly"},
		actions: []string{"compile: notestfiles", "link: cgomain", "pack: cgoonlynotest", "pack: cgotest"},
	}}

	for _, tt := range tests {
		ctx := testContext(t)
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
		if !sameErr(err, tt.err) {
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

func TestPkgname(t *testing.T) {
	tests := []struct {
		pkg  string
		name string
	}{{
		pkg:  "a",
		name: "a",
	}, {
		pkg:  "b",
		name: "b",
	}}

	ctx := testContext(t)
	defer ctx.Destroy()
	for _, tt := range tests {
		pkg, err := ctx.ResolvePackage(tt.pkg)
		if err != nil {
			t.Error(err)
			continue
		}
		if got, want := pkgname(pkg), tt.name; got != want {
			t.Errorf("pkgname(%v): want %v, got %v", want, got)
		}
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
