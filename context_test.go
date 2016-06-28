package gb

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/constabulary/gb/internal/importer"
)

func testImportCycle(pkg string, t *testing.T) {
	ctx := testContext(t)

	debug.SetMaxStack(1 << 18)

	_, err := ctx.ResolvePackage(pkg)
	if strings.Index(err.Error(), "cycle detected") == -1 {
		t.Errorf("ctx.ResolvePackage returned wrong error. Expected cycle detection, got: %v", err)
	}

	if err == nil {
		t.Errorf("ctx.ResolvePackage should have returned an error for cycle, returned nil")
	}
}

func TestOneElementCycleDetection(t *testing.T) {
	testImportCycle("cycle0", t)
}

func TestSimpleCycleDetection(t *testing.T) {
	testImportCycle("cycle1/a", t)
}

func TestLongCycleDetection(t *testing.T) {
	testImportCycle("cycle2/a", t)
}

func TestContextCtxString(t *testing.T) {
	opts := func(o ...func(*Context) error) []func(*Context) error { return o }
	join := func(s ...string) string { return strings.Join(s, "-") }
	tests := []struct {
		opts []func(*Context) error
		want string
	}{
		{nil, join(runtime.GOOS, runtime.GOARCH)},
		{opts(GOOS("windows")), join("windows", runtime.GOARCH)},
		{opts(GOARCH("arm64")), join(runtime.GOOS, "arm64")},
		{opts(GOARCH("arm64"), GOOS("plan9")), join("plan9", "arm64")},
		{opts(Tags()), join(runtime.GOOS, runtime.GOARCH)},
		{opts(Tags("sphinx", "leon")), join(runtime.GOOS, runtime.GOARCH, "leon", "sphinx")},
		{opts(Tags("sphinx", "leon"), GOARCH("ppc64le")), join(runtime.GOOS, "ppc64le", "leon", "sphinx")},
	}

	proj := testProject(t)
	for _, tt := range tests {
		ctx, err := NewContext(proj, tt.opts...)
		if err != nil {
			t.Fatal(err)
		}
		defer ctx.Destroy()
		got := ctx.ctxString()
		if got != tt.want {
			t.Errorf("NewContext(%q).ctxString(): got %v, want %v", tt.opts, got, tt.want)
		}
	}
}

func TestContextOptions(t *testing.T) {
	matches := func(want Context) func(t *testing.T, got *Context) {
		return func(t *testing.T, got *Context) {
			if !reflect.DeepEqual(got, &want) {
				t.Errorf("got %#v, want %#v", got, &want)
			}
		}
	}

	tests := []struct {
		ctx    Context
		fn     func(*Context) error
		err    error
		expect func(*testing.T, *Context)
	}{{
		// assert that an zero context is not altered by the test rig.
		fn:     func(*Context) error { return nil },
		expect: matches(Context{}),
	}, {
		// test blank GOOS is an error
		fn:  GOOS(""),
		err: fmt.Errorf("GOOS cannot be blank"),
	}, {
		// test blank GOARCH is an error
		fn:  GOARCH(""),
		err: fmt.Errorf("GOARCH cannot be blank"),
	}, {
		ctx: Context{
			gotargetos:   "bar",
			gotargetarch: "baz",
		},
		fn: GOOS("foo"),
		expect: matches(Context{
			gotargetos:   "foo",
			gotargetarch: "baz",
		}),
	}, {
		ctx: Context{
			gotargetos:   "bar",
			gotargetarch: "baz",
		},
		fn: GOARCH("foo"),
		expect: matches(Context{
			gotargetos:   "bar",
			gotargetarch: "foo",
		}),
	}, {
		fn:     Tags(),
		expect: matches(Context{}),
	}, {
		fn:     Tags("foo"),
		expect: matches(Context{buildtags: []string{"foo"}}),
	}, {
		ctx:    Context{buildtags: []string{"foo"}},
		fn:     Tags("bar"),
		expect: matches(Context{buildtags: []string{"foo", "bar"}}),
	}, {
		fn:     Gcflags("foo"),
		expect: matches(Context{gcflags: []string{"foo"}}),
	}, {
		ctx:    Context{gcflags: []string{"foo"}},
		fn:     Gcflags("bar"),
		expect: matches(Context{gcflags: []string{"foo", "bar"}}),
	}, {
		fn:     Ldflags("foo"),
		expect: matches(Context{ldflags: []string{"foo"}}),
	}, {
		ctx:    Context{ldflags: []string{"foo"}},
		fn:     Ldflags("bar"),
		expect: matches(Context{ldflags: []string{"foo", "bar"}}),
	}, {
		fn: WithRace,
		expect: matches(Context{
			buildtags: []string{"race"},
			race:      true,
			gcflags:   []string{"-race"},
			ldflags:   []string{"-race"},
		}),
	}, {
		ctx: Context{buildtags: []string{"zzz"}},
		fn:  WithRace,
		expect: matches(Context{
			buildtags: []string{"zzz", "race"},
			race:      true,
			gcflags:   []string{"-race"},
			ldflags:   []string{"-race"},
		}),
	}}

	for i, tt := range tests {
		ctx := tt.ctx
		err := tt.fn(&ctx)
		switch {
		case !reflect.DeepEqual(err, tt.err):
			t.Errorf("test %d: expected err: %v, got %v", i+1, tt.err, err)
		case err == nil:
			tt.expect(t, &ctx)
		}
	}
}

func TestContextLoadPackage(t *testing.T) {
	tests := []struct {
		opts []func(*Context) error
		path string
	}{
		{path: "errors"},
		{path: "net/http"}, // triggers vendor logic on go 1.6+
	}

	proj := testProject(t)
	for _, tt := range tests {
		ctx, err := NewContext(proj, tt.opts...)
		if err != nil {
			t.Fatal(err)
		}
		defer ctx.Destroy()
		_, err = ctx.loadPackage(nil, tt.path)
		if err != nil {
			t.Errorf("loadPackage(%q): %v", tt.path, err)
		}
	}
}

func TestCgoEnabled(t *testing.T) {
	tests := []struct {
		gohostos, gohostarch     string
		gotargetos, gotargetarch string
		want                     bool
	}{{
		"linux", "amd64", "linux", "amd64", true,
	}, {
		"linux", "amd64", "linux", "386", false,
	}}

	for _, tt := range tests {
		got := cgoEnabled(tt.gohostos, tt.gohostarch, tt.gotargetos, tt.gotargetarch)
		if got != tt.want {
			t.Errorf("cgoEnabled(%q, %q, %q, %q): got %v, want %v", tt.gohostos, tt.gohostarch, tt.gotargetos, tt.gotargetarch, got, tt.want)
		}
	}
}

func TestContextImportPackage(t *testing.T) {
	proj := testProject(t)
	tests := []struct {
		path string
		err  error
	}{{
		path: "a",
	}, {
		path: "cgomain",
	}, {
		path: "net/http", // loaded from GOROOT
	}, {
		path: "cmd",
		err:  &importer.NoGoError{Dir: filepath.Join(proj.Projectdir(), "src", "cmd")},
	}}

	for _, tt := range tests {
		ctx, err := NewContext(proj)
		if err != nil {
			t.Fatal(err)
		}
		_, err = ctx.importer.Import(tt.path)
		if !reflect.DeepEqual(err, tt.err) {
			t.Errorf("importPackage(%q): got %v, want %v", tt.path, err, tt.err)
		}
	}
}
