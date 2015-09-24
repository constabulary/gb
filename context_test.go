package gb

import (
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
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
		ctx, err := proj.NewContext(tt.opts...)
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
