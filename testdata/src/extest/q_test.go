package extest

import "testing"

func TestV(t *testing.T) {
	if V != 0 {
		t.Fatalf("V: got %v, expected 0", V)
	}
}
