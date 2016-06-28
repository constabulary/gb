package gb

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

type testproject struct {
	*testing.T
	project
}

func testProject(t *testing.T) Project {
	cwd := getwd(t)
	root := filepath.Join(cwd, "testdata")
	return &testproject{
		t,
		project{
			rootdir: root,
		},
	}
}

func tempProject(t *testing.T) *testproject {
	return &testproject{
		t,
		project{
			rootdir: mktemp(t),
		},
	}
}

func (t *testproject) tempfile(path, contents string) string {
	dir, file := filepath.Split(path)
	dir = filepath.Join(t.rootdir, dir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path = filepath.Join(dir, file)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(f, contents); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}
