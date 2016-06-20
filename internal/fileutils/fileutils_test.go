package fileutils

import (
	"io/ioutil"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCopypathSkipsSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no symlinks on windows y'all")
	}
	dst := mktemp(t)
	defer RemoveAll(dst)
	src := filepath.Join("_testdata", "copyfile", "a")
	if err := Copypath(dst, src); err != nil {
		t.Fatalf("copypath(%s, %s): %v", dst, src, err)
	}
}

func mktemp(t *testing.T) string {
	s, err := ioutil.TempDir("", "fileutils_test")
	if err != nil {
		t.Fatal(err)
	}
	return s
}
