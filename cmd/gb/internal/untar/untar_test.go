package untar

import (
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestUntar(t *testing.T) {
	tests := []struct {
		desc string
		r    func(*testing.T) io.ReadCloser
		dest string
		want []string
	}{{
		desc: "a.tar.gz",
		r:    tgz("_testdata/a.tar.gz"),
		want: []string{
			".",
			"a",
			"a/a",
		},
	}, {
		desc: "errors.tar.gz",
		r:    tgz("_testdata/errors.tar.gz"),
		want: []string{
			".",
			"pkg-errors-805fb19",
			"pkg-errors-805fb19/.gitignore",
			"pkg-errors-805fb19/.travis.yml",
			"pkg-errors-805fb19/LICENSE",
			"pkg-errors-805fb19/README.md",
			"pkg-errors-805fb19/appveyor.yml",
			"pkg-errors-805fb19/errors.go",
			"pkg-errors-805fb19/errors_test.go",
			"pkg-errors-805fb19/example_test.go",
			"pkg-errors-805fb19/format_test.go",
			"pkg-errors-805fb19/stack.go",
			"pkg-errors-805fb19/stack_test.go",
		},
	}, {
		desc: "symlink.tar.gz",
		r:    tgz("_testdata/symlink.tar.gz"),
		want: []string{
			".",
			"symlink",
			"symlink/a",
			// no symlink/b
		},
	}}

	for _, tt := range tests {
		dest := tmpdir(t)
		defer os.RemoveAll(dest)
		r := tt.r(t)
		defer r.Close()
		err := Untar(dest, r)
		if err != nil {
			t.Error(err)
			continue
		}
		got := walkdir(t, dest)
		want := make([]string, len(tt.want))
		for i := range tt.want {
			want[i] = filepath.FromSlash(tt.want[i])
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: untar: expected %s, got %s", tt.desc, want, got)
		}
	}
}

func tmpdir(t *testing.T) string {
	dir, err := ioutil.TempDir("", ".test")
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, "dest")
}

func tgz(path string) func(t *testing.T) io.ReadCloser {
	return func(t *testing.T) io.ReadCloser {
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		r, err := gzip.NewReader(f)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
}

func walkdir(t *testing.T, root string) []string {
	var paths []string
	err := filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		path, err = filepath.Rel(root, path)
		paths = append(paths, path)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return paths
}
