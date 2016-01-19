package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"syscall"
	"testing"
)

func TestImporter(t *testing.T) {
	patherr := func(path string) error {
		op := "stat"
		if runtime.GOOS == "windows" {
			op = "GetFileAttributesEx"
		}
		return &os.PathError{
			Op:   op,
			Path: path,
			Err:  syscall.ENOENT,
		}
	}
	tests := []struct {
		Importer
		path string
		want *Package
		err  error
	}{{
		Importer: Importer{},
		path:     "",
		err:      fmt.Errorf("import %q: invalid import path", ""),
	}, {
		Importer: Importer{},
		path:     ".",
		err:      fmt.Errorf("import %q: relative import not supported", "."),
	}, {
		Importer: Importer{},
		path:     "..",
		err:      fmt.Errorf("import %q: relative import not supported", ".."),
	}, {
		Importer: Importer{},
		path:     "./",
		err:      fmt.Errorf("import %q: relative import not supported", "./"),
	}, {
		Importer: Importer{},
		path:     "../",
		err:      fmt.Errorf("import %q: relative import not supported", "../"),
	}, {
		Importer: Importer{},
		path:     "/foo",
		err:      fmt.Errorf("import %q: cannot import absolute path", "/foo"),
	}, {
		Importer: Importer{
			Context: &Context{
				GOOS:   "linux",
				GOARCH: "amd64",
			},
			Root: filepath.Join(runtime.GOROOT()),
		},
		path: "errors",
		want: &Package{
			ImportPath: "errors",
			Name:       "errors", // no yet
			Dir:        filepath.Join(runtime.GOROOT(), "src", "errors"),
			Root:       filepath.Join(runtime.GOROOT()),
			SrcRoot:    filepath.Join(runtime.GOROOT(), "src"),
			Standard:   true,

			GoFiles:      []string{"errors.go"},
			XTestGoFiles: []string{"errors_test.go", "example_test.go"},
			XTestImports: []string{"errors", "fmt", "testing", "time"},
		},
	}, {
		Importer: Importer{
			Context: &Context{
				GOOS:   "linux",
				GOARCH: "amd64",
			},
			Root: filepath.Join(runtime.GOROOT()),
		},
		path: "database",
		err:  &NoGoError{filepath.Join(runtime.GOROOT(), "src", "database")},
	}, {
		Importer: Importer{
			Context: &Context{
				GOOS:   "linux",
				GOARCH: "amd64",
			},
			Root: filepath.Join(runtime.GOROOT()),
		},
		path: "missing",
		err:  patherr(filepath.Join(runtime.GOROOT(), "src", "missing")),
	}, {
		Importer: Importer{
			Context: &Context{
				GOOS:       "linux",
				GOARCH:     "amd64",
				CgoEnabled: true,
			},
			Root: filepath.Join(runtime.GOROOT()),
		},
		path: "net",
	}, {
		Importer: Importer{
			Context: &Context{
				GOOS:       "linux",
				GOARCH:     "amd64",
				CgoEnabled: true,
			},
			Root: filepath.Join(runtime.GOROOT()),
		},
		path: "os/user",
	}}

	for _, tt := range tests {
		got, err := tt.Import(tt.path)
		if !reflect.DeepEqual(err, tt.err) {
			t.Errorf("Import(%q): got err %v, want err %v", tt.path, err, tt.err)
		}

		if err != nil {
			continue
		}

		if tt.want == nil {
			t.Logf("Import(%q): skipping package contents check", tt.path)
			continue
		}

		// fixups
		want := tt.want
		want.importer = got.importer
		got.ImportPos = nil
		got.TestImportPos = nil
		got.XTestImportPos = nil

		if !reflect.DeepEqual(got, want) {
			t.Errorf("Import(%q): got %#v, want %#v", tt.path, got, want)
		}
	}
}
