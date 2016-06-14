package fileutils

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCopypathSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no symlinks on windows y'all")
	}
	dst := mktemp(t)
	defer RemoveAll(dst)
	src := filepath.Join("_testdata", "copyfile")
	if err := Copypath(dst, src); err != nil {
		t.Fatalf("copypath(%s, %s): %v", dst, src, err)
	}
	res, err := os.Readlink(filepath.Join(dst, "a", "rick"))
	if err != nil {
		t.Fatal(err)
	}
	if res != "/never/going/to/give/you/up" {
		t.Fatalf("target == %s, expected /never/going/to/give/you/up", res)
	}
}

func mktemp(t *testing.T) string {
	s, err := ioutil.TempDir("", "fileutils_test")
	if err != nil {
		t.Fatal(err)
	}
	return s
}
