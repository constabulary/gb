package gb

import (
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
