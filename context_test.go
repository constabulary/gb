package gb

import (
	"strings"
	"testing"
	"time"
)

func testImportCycle(pkg string, t *testing.T) {
	ctx := testContext(t)

	// one of two goroutines should return to finish the test
	done := make(chan bool, 2)

	timeConstraint := time.AfterFunc(1*time.Second, func() {
		t.Error("ctx.ResolvePackage have not finished in 1s")
		done <- true
	})

	go func() {
		_, err := ctx.ResolvePackage(pkg)
		timeConstraint.Stop()

		if strings.Index(err.Error(), "cycle detected") == -1 {
			t.Errorf("ctx.ResolvePackage returned wrong error. Expected cycle detection, got: %v", err)
		}

		if err == nil {
			t.Errorf("ctx.ResolvePackage should have returned an error for cycle, returned nil")
		}
		done <- true
	}()

	<-done
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
