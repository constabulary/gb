package gb

import (
	"fmt"
	"go/build"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func testProject(t *testing.T) *Project {
	cwd := getwd(t)
	root := filepath.Join(cwd, "testdata")
	return NewProject(root,
		SourceDir(filepath.Join(root, "src")),
	)
}

func testContext(t *testing.T, opts ...func(*Context) error) *Context {
	prj := testProject(t)
	ctx, err := prj.NewContext(opts...)
	if err != nil {
		t.Fatal(err)
	}
	ctx.Force = true
	ctx.SkipInstall = true
	return ctx
}

func TestResolvePackage(t *testing.T) {
	ctx := testContext(t)
	defer ctx.Destroy()
	_, err := ctx.ResolvePackage("a")
	if err != nil {
		t.Fatal(err)
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
		ctx, _ := proj.NewContext(tt.opts...)
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
			Package: &build.Package{
				Name:       "main",
				ImportPath: "main",
			},
		},
		want: true,
	}, {
		pkg: &Package{
			Package: &build.Package{
				Name:       "a",
				ImportPath: "main",
			},
		},
		want: false,
	}, {
		pkg: &Package{
			Package: &build.Package{
				Name:       "main",
				ImportPath: "a",
			},
		},
		want: true,
	}, {
		pkg: &Package{
			Package: &build.Package{
				Name:       "main",
				ImportPath: "testmain",
			},
		},
		want: true,
	}, {
		pkg: &Package{
			Package: &build.Package{
				Name:       "main",
				ImportPath: "main",
			},
			Scope: "test",
		},
		want: false,
	}, {
		pkg: &Package{
			Package: &build.Package{
				Name:       "a",
				ImportPath: "main",
			},
			Scope: "test",
		},
		want: false,
	}, {
		pkg: &Package{
			Package: &build.Package{
				Name:       "main",
				ImportPath: "a",
			},
			Scope: "test",
		},
		want: false,
	}, {
		pkg: &Package{
			Package: &build.Package{
				Name:       "main",
				ImportPath: "testmain",
			},
			Scope: "test",
		},
		want: true,
	}}

	for _, tt := range tests {
		got := tt.pkg.isMain()
		if got != tt.want {
			t.Errorf("Package{Name:%q, ImportPath: %q, Scope:%q}.isMain(): got %v, want %v", tt.pkg.Name, tt.pkg.ImportPath, tt.pkg.Scope, got, tt.want)
		}
	}
}

func TestNewPackage(t *testing.T) {
	tests := []struct {
		pkg  build.Package
		want Package
	}{{
		build.Package{
			Name:       "C",
			ImportPath: "C",
			Goroot:     true,
		},
		Package{
			Stale:    false,
			Standard: true,
		},
	}}
	proj := testProject(t)
	for i, tt := range tests {
		ctx, _ := proj.NewContext()
		defer ctx.Destroy()

		got := NewPackage(ctx, &tt.pkg)
		want := tt.want // deep copy
		want.Package = &tt.pkg
		want.Context = ctx

		if !reflect.DeepEqual(got, &want) {
			t.Errorf("%d: pkg: %s: expected %#v, got %#v", i+1, tt.pkg.ImportPath, &want, got)
		}
	}
}
