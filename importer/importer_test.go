package importer

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestImporter(t *testing.T) {
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
	}}

	for _, tt := range tests {
		got, err := tt.Import(tt.path)
		if !reflect.DeepEqual(err, tt.err) {
			t.Errorf("Import(%q): got err %v, want err %v", tt.path, err, tt.err)
		}

		if err != nil {
			continue
		}

		// fixups
		want := tt.want
		want.Importer = got.Importer
		got.ImportPos = nil
		got.TestImportPos = nil
		got.XTestImportPos = nil

		if !reflect.DeepEqual(got, want) {
			t.Errorf("Import(%q): got %#v, want %#v", tt.path, got, want)
		}
	}
}
