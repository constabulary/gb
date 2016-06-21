package test

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/constabulary/gb"
)

func TestTest(t *testing.T) {
	tests := []struct {
		pkg        string
		testArgs   []string
		ldflags    []string
		err        error
		minversion float64 // minimum go version that supports this feature
		maxversion float64 // maximum go version that supports this feature
	}{
		{
			pkg: "a",
			err: nil,
		}, {
			pkg: "b",
			err: nil,
		}, {
			pkg: "c",
			err: nil,
		}, {
			pkg: "e",
			err: nil,
		}, {
			pkg: "cmd/f",
			err: nil,
		}, {
			pkg: "extest", // test external tests
			err: nil,
		}, {
			pkg: "external_only_test", // issue 312
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
			pkg: "g", // test that _test files can modify the internal package under test
			err: nil,
		}, {
			pkg:        "ldflags",
			ldflags:    []string{"-X", "ldflags.gitTagInfo=banana", "-X", "ldflags.gitRevision=f7926af2"},
			minversion: 1.5,
		}, {
			pkg:        "ldflags",
			ldflags:    []string{"-X", "ldflags.gitTagInfo", "banana", "-X", "ldflags.gitRevision", "f7926af2"},
			maxversion: 1.5,
		}, {
			pkg: "cgotest",
		}, {
			pkg:      "testflags",
			testArgs: []string{"-debug"},
		}, {
			pkg: "main", // issue 375, a package called main
		}}

	for _, tt := range tests {
		if tt.minversion != 0 && goversion < tt.minversion {
			t.Logf("skipping test, goversion %f is below mingoversion %f", goversion, tt.minversion)
			continue
		}
		if tt.maxversion != 0 && goversion > tt.maxversion {
			t.Logf("skipping test, goversion %f is above maxgoversion %f", goversion, tt.maxversion)
			continue
		}
		ctx := testContext(t, gb.Ldflags(tt.ldflags...))
		defer ctx.Destroy()
		r := TestResolver(ctx)
		pkg, err := r.ResolvePackage(tt.pkg)
		if err != nil {
			t.Errorf("ResolvePackage(%v): want %v, got %v", tt.pkg, tt.err, err)
			continue
		}
		if err := Test(tt.testArgs, pkg); err != tt.err {
			t.Errorf("Test(%v): want %v, got %v", tt.pkg, tt.err, err)
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func TestTestPackage(t *testing.T) {
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
		err: nil,
	}, {
		pkg: "extestonly",
		err: nil,
	}}

	for _, tt := range tests {
		ctx := testContext(t)
		defer ctx.Destroy()
		r := TestResolver(ctx)
		pkg, err := r.ResolvePackage(tt.pkg)
		if err != nil {
			t.Errorf("r.ResolvePackage(%v):  %v", tt.pkg, err)
			continue
		}
		targets := make(map[string]*gb.Action)
		if _, err := TestPackage(targets, pkg, nil); !reflect.DeepEqual(err, tt.err) {
			t.Errorf("TestPackage(%v): want %v, got %v", tt.pkg, tt.err, err)
		}
	}
}

func TestTestPackages(t *testing.T) {
	tests := []struct {
		pkgs    []string
		actions []string
		err     error
	}{{
		pkgs: []string{"a", "b", "c"},
		actions: []string{
			"run: $WORKDIR/a/testmain/_test/a.test$EXE",
			"run: $WORKDIR/b/testmain/_test/b.test$EXE",
			"run: $WORKDIR/c/testmain/_test/c.test$EXE",
		},
	}, {
		pkgs: []string{"cgotest", "cgomain", "notestfiles", "cgoonlynotest", "testonly", "extestonly"},
		actions: []string{
			"run: $WORKDIR/cgomain/testmain/_test/cgomain.test$EXE",
			"run: $WORKDIR/cgoonlynotest/testmain/_test/cgoonly.test$EXE",
			"run: $WORKDIR/cgotest/testmain/_test/cgotest.test$EXE",
			"run: $WORKDIR/extestonly/testmain/_test/extestonly.test$EXE",
			"run: $WORKDIR/notestfiles/testmain/_test/notest.test$EXE",
			"run: $WORKDIR/testonly/testmain/_test/testonly.test$EXE",
		},
	}}

	for i, tt := range tests {
		ctx := testContext(t)
		defer ctx.Destroy()
		var pkgs []*gb.Package
		t.Logf("testing: %v: pkgs: %v", i+1, tt.pkgs)
		r := TestResolver(ctx)
		for _, pkg := range tt.pkgs {
			pkg, err := r.ResolvePackage(pkg)
			if err != nil {
				t.Errorf("r.ResolvePackage(%v):  %v", pkg, err)
				continue
			}
			pkgs = append(pkgs, pkg)
		}
		a, err := TestPackages(nil, pkgs...)
		if !reflect.DeepEqual(err, tt.err) {
			t.Errorf("TestPackages(%v): want %v, got %v", pkgs, tt.err, err)
		}
		var actual []string
		for _, a := range a.Deps {
			actual = append(actual, a.Name)
		}
		sort.Strings(actual)
		var expected []string
		exe := ""
		if runtime.GOOS == "windows" {
			exe = ".exe"
		}
		for _, s := range tt.actions {
			s = filepath.FromSlash(s)
			s = strings.Replace(s, "$WORKDIR", ctx.Workdir(), -1)
			s = strings.Replace(s, "$EXE", exe, -1)
			expected = append(expected, s)
		}
		if !reflect.DeepEqual(expected, actual) {
			t.Errorf("TestBuildPackages(%v): want %v, got %v", pkgs, expected, actual)
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

func testProject(t *testing.T) gb.Project {
	cwd := getwd(t)
	root := filepath.Join(cwd, "..", "testdata")
	return gb.NewProject(root)
}

func testContext(t *testing.T, opts ...func(*gb.Context) error) *gb.Context {
	opts = append([]func(*gb.Context) error{gb.GcToolchain()}, opts...)
	ctx, err := gb.NewContext(testProject(t), opts...)
	if err != nil {
		t.Fatal(err)
	}
	ctx.Force = true
	return ctx
}
