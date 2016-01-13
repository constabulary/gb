package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
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

func TestFindProjectroot(t *testing.T) {
	root := makeTestdata(t)
	defer os.RemoveAll(root)
	tests := []struct {
		path string
		want string
		err  error
	}{
		{path: root, want: root},
		{path: join(root, "src"), want: root},
		{path: join(join(root, "src"), "a"), want: root},
		{path: join(root, ".."), err: fmt.Errorf(`could not find project root in "%s" or its parents`, join(root, ".."))},
	}

	for _, tt := range tests {
		got, err := FindProjectroot(tt.path)
		if got != tt.want || !reflect.DeepEqual(err, tt.err) {
			t.Errorf("FindProjectroot(%v): want: %v, %v, got %v, %v", tt.path, tt.want, tt.err, got, err)
		}
	}
}
