package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

var join = filepath.Join

// makeTestData constructs
func makeTestdata(t *testing.T) string {
	tempdir, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	root, err := ioutil.TempDir(tempdir, "path-test")
	if err != nil {
		t.Fatal(err)
	}
	mkdir := func(args ...string) string {
		path := join(args...)
		if err := os.MkdirAll(path, 0777); err != nil {
			t.Fatal(err)
		}
		return path
	}
	mkfile := func(path string, content string) {
		if err := ioutil.WriteFile(path, []byte(content), 0755); err != nil {
			t.Fatal(err)
		}
	}

	srcdir := mkdir(root, "src")
	mkfile(join(mkdir(srcdir, "a"), "a.go"), "package a")

	return root
}

func TestRelImportPath(t *testing.T) {
	tests := []struct {
		root, path, want string
	}{
		{"/project/src", "a", "a"},
		// { "/project/src", "./a", "a"}, // TODO(dfc) this is relative
		// { "/project/src", "a/../b", "a"}, // TODO(dfc) so is this
	}

	for _, tt := range tests {

		got := relImportPath(tt.root, tt.path)
		if got != tt.want {
			t.Errorf("relImportPath(%q, %q): want: %v, got: %v", tt.root, tt.path, tt.want, got)
		}
	}
}

func TestIsRel(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{".", true},
		{"..", false},     // TODO(dfc) this is relative
		{"a/../b", false}, // TODO(dfc) this too
	}

	for _, tt := range tests {
		got := isRel(tt.path)
		if got != tt.want {
			t.Errorf("isRel(%q): want: %v, got: %v", tt.want, got)
		}
	}
}
