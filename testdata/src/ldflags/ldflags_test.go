package ldflags

import "testing"

var gitTagInfo string // set by linker

func TestLdflags(t *testing.T) {
	if gitTagInfo != "banana" {
		t.Error("gitTagInfo:", gitTagInfo)
	}
	if gitRevision != "f7926af2" {
		t.Error("gitRevision:", gitRevision)
	}
}
