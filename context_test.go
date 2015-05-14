package gb

import (
	"testing"
	"time"
)

func TestCycleDetection(t *testing.T) {
	ctx := testContext(t)

	// one of two goroutines should return to finish the test
	done := make(chan bool, 2)

	timeConstraint := time.AfterFunc(1*time.Second, func() {
		t.Error("ctx.ResolvePackage have not finished in 1s")
		done <- true
	})

	go func() {
		_, err := ctx.ResolvePackage("cycle1.a")
		timeConstraint.Stop()
		if err == nil {
			t.Errorf("ctx.ResolvePackage should have returned an error for cycle, returned nil")
		}
		done <- true
	}()

	<-done
}
