package vendor

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCopypathSkipsSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no symlinks on windows y'all")
	}
	dst := mktemp(t)
	defer os.RemoveAll(dst)
	src := filepath.Join("_testdata", "copyfile", "a")
	if err := Copypath(dst, src); err != nil {
		t.Fatalf("copypath(%s, %s): %v", dst, src, err)
	}
}
