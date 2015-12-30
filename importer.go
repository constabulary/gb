package gb

import (
	"go/build"
	"path/filepath"
	"runtime"
)

type importer struct {
	srcdir string
	bc     *build.Context
}

// Import imports the package indicated by the import path.
func (i *importer) Import(path string) (*build.Package, error) {
	// build.Context.Import takes a directory argument that is relative to
	// a specific directory. This is to support relative imports and vendoring.
	// gb does not support either of these in the project, but we must do so
	// to support recursing into the stdlib and rebuilding it.
	//
	// It would be great to ignore vendoring completely, but that means we cannot
	// cross compile the 1.6+ stdlib which uses vendoring for http2. So, we do a
	// horrid hack.
	mode := build.ImportMode(0)
	dir := i.srcdir
	if goversion > 1.5 && path == "golang.org/x/net/http2/hpack" {
		mode |= allowVendor
		dir = filepath.Join(runtime.GOROOT(), "src", "net", "http")
	}
	return i.bc.Import(path, dir, mode)
}
