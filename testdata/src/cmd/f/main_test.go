package main

import "testing"

func TestX(t *testing.T) {
	if X != 7 {
		t.Fatal("X != 7")
	}
}
