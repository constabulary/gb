package gb

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/constabulary/gb/internal/importer"
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
		err: fmt.Errorf(`import "../localimport": relative import not supported`),
	}}
	proj := testProject(t)
	for _, tt := range tests {
		ctx, err := NewContext(proj, tt.opts...)
		defer ctx.Destroy()
		_, err = ctx.ResolvePackage(tt.pkg)
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
		want := filepath.Join(ctx.Bindir(), tt.want)
		if pkg.gotargetos == "windows" {
			want += ".exe"
		}
		if want != got {
			t.Errorf("test %v: (%s).Binfile(): want %s, got %s", i+1, tt.pkg, want, got)
		}
	}
}

func TestPackageIsMain(t *testing.T) {
	var tests = []struct {
		pkg  *Package
		want bool
	}{{
		pkg: &Package{
			Package: &importer.Package{
				Name:       "main",
				ImportPath: "main",
			},
		},
		want: true,
	}, {
		pkg: &Package{
			Package: &importer.Package{
				Name:       "a",
				ImportPath: "main",
			},
		},
		want: false,
	}, {
		pkg: &Package{
			Package: &importer.Package{
				Name:       "main",
				ImportPath: "a",
			},
		},
		want: true,
	}, {
		pkg: &Package{
			Package: &importer.Package{
				Name:       "main",
				ImportPath: "testmain",
			},
		},
		want: true,
	}, {
		pkg: &Package{
			Package: &importer.Package{
				Name:       "main",
				ImportPath: "main",
			},
			TestScope: true,
		},
		want: false,
	}, {
		pkg: &Package{
			Package: &importer.Package{
				Name:       "a",
				ImportPath: "main",
			},
			TestScope: true,
		},
		want: false,
	}, {
		pkg: &Package{
			Package: &importer.Package{
				Name:       "main",
				ImportPath: "a",
			},
			TestScope: true,
		},
		want: false,
	}, {
		pkg: &Package{
			Package: &importer.Package{
				Name:       "main",
				ImportPath: "testmain",
			},
			TestScope: true,
		},
		want: true,
	}}

	for _, tt := range tests {
		got := tt.pkg.isMain()
		if got != tt.want {
			t.Errorf("Package{Name:%q, ImportPath: %q, TestScope:%v}.isMain(): got %v, want %v", tt.pkg.Name, tt.pkg.ImportPath, tt.pkg.TestScope, got, tt.want)
		}
	}
}

func TestNewPackage(t *testing.T) {
	tests := []struct {
		pkg  importer.Package
		want Package
	}{{
		importer.Package{
			Name:       "C",
			ImportPath: "C",
			Standard:   true,
		},
		Package{
			Stale: false,
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
