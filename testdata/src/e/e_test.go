package e

import "testing"
import "f" // only imported in internal test scope

func TestE(t *testing.T) {
	t.Log(f.F > 0.9)
}
