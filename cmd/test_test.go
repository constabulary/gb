package cmd

import (
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/log"
)

func TestTest(t *testing.T) {
	log.Verbose = false
	defer func() { log.Verbose = false }()
	tests := []struct {
		pkg      string
		testArgs []string
		ldflags  string
		err      error
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
			pkg:     "ldflags",
			ldflags: "-X ldflags.gitTagInfo banana -X ldflags.gitRevision f7926af2",
		}, {
			pkg: "cgotest",
		}, {
			pkg:      "testflags",
			testArgs: []string{"-debug"},
		}, {
			pkg: "main", // issue 375, a package called main
		}}

	for _, tt := range tests {
		ctx := testContext(t, gb.Ldflags(tt.ldflags))
		defer ctx.Destroy()
		// TODO(dfc) can we resolve the duplication here ?
		pkg, err := ctx.ResolvePackageWithTests(tt.pkg)
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
		pkg, err := ctx.ResolvePackage(tt.pkg)
		if err != nil {
			t.Errorf("ctx.ResolvePackage(%v):  %v", tt.pkg, err)
			continue
		}
		targets := make(map[string]*gb.Action)
		if _, err := TestPackage(targets, pkg, nil); !sameErr(err, tt.err) {
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
		pkgs:    []string{"a", "b", "c"},
		actions: []string{"run: [$WORKDIR/a/testmain/_test/a.test]", "run: [$WORKDIR/b/testmain/_test/b.test]", "run: [$WORKDIR/c/testmain/_test/c.test]"},
	}, {
		pkgs:    []string{"cgotest", "cgomain", "notestfiles", "cgoonlynotest", "testonly", "extestonly"},
		actions: []string{"run: [$WORKDIR/cgomain/testmain/_test/cgomain.test]", "run: [$WORKDIR/cgoonlynotest/testmain/_test/cgoonly.test]", "run: [$WORKDIR/cgotest/testmain/_test/cgotest.test]", "run: [$WORKDIR/extestonly/testmain/_test/extestonly.test]", "run: [$WORKDIR/notestfiles/testmain/_test/notest.test]", "run: [$WORKDIR/testonly/testmain/_test/testonly.test]"},
	}}

	for _, tt := range tests {
		ctx := testContext(t)
		defer ctx.Destroy()
		var pkgs []*gb.Package
		for _, pkg := range tt.pkgs {
			pkg, err := ctx.ResolvePackage(pkg)
			if err != nil {
				t.Errorf("ctx.ResolvePackage(%v):  %v", pkg, err)
				continue
			}
			pkgs = append(pkgs, pkg)
		}
		a, err := TestPackages(nil, pkgs...)
		if !sameErr(err, tt.err) {
			t.Errorf("TestPackages(%v): want %v, got %v", pkgs, tt.err, err)
		}
		var actual []string
		for _, a := range a.Deps {
			actual = append(actual, a.Name)
		}
		sort.Strings(actual)
		var expected []string
		for _, s := range tt.actions {
			expected = append(expected, strings.Replace(s, "$WORKDIR", ctx.Workdir(), -1))
		}
		if !reflect.DeepEqual(expected, actual) {
			t.Errorf("TestBuildPackages(%v): want %v, got %v", pkgs, expected, actual)
		}
	}
}
