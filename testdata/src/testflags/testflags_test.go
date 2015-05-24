package testflags

import (
	"flag"
	"testing"
)

var debug bool

func init() {
	flag.BoolVar(&debug, "debug", false, "Enable debug output.")
	flag.Parse()
}

func TestDebug(t *testing.T) {
	if !debug {
		t.Error("debug not true!")
	}
}
