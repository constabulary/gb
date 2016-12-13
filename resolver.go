package gb

import (
	"fmt"
	"go/build"
	"os"
	pathpkg "path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"
)

type nullImporter struct{}

func (i *nullImporter) Import(path string) (*build.Package, error) {
	return nil, errors.Errorf("import %q not found", path)
}

type srcImporter struct {
	Importer
	im importer
}

func (i *srcImporter) Import(path string) (*build.Package, error) {
	pkg, err := i.im.Import(path)
	if err == nil {
		return pkg, nil
	}

	// gb expects, when there is a failure to resolve packages that
	// live in $PROJECT/src that the importer for that directory
	// will report them.

	pkg, err2 := i.Importer.Import(path)
	if err2 == nil {
		return pkg, nil
	}
	return nil, err
}

type _importer struct {
	Importer
	im importer
}

func (i *_importer) Import(path string) (*build.Package, error) {
	pkg, err := i.im.Import(path)
	if err != nil {
		return i.Importer.Import(path)
	}
	return pkg, nil
}

type fixupImporter struct {
	Importer
}

func (i *fixupImporter) Import(path string) (*build.Package, error) {
	pkg, err := i.Importer.Import(path)
	switch err.(type) {
	case *os.PathError:
		return nil, errors.Wrapf(err, "import %q: not found", path)
	default:
		return pkg, err
	}
}

type importer struct {
	*build.Context
	Root string // root directory
}

type importErr struct {
	path string
	msg  string
}

func (e *importErr) Error() string {
	return fmt.Sprintf("import %q: %v", e.path, e.msg)
}

func (i *importer) Import(path string) (*build.Package, error) {
	if path == "" {
		return nil, errors.WithStack(&importErr{path: path, msg: "invalid import path"})
	}

	if path == "." || path == ".." || strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") {
		return nil, errors.WithStack(&importErr{path: path, msg: "relative import not supported"})
	}

	if strings.HasPrefix(path, "/") {
		return nil, errors.WithStack(&importErr{path: path, msg: "cannot import absolute path"})
	}

	var p *build.Package

	loadPackage := func(importpath, dir string) error {
		pkg, err := i.ImportDir(dir, 0)
		if err != nil {
			return err
		}
		p = pkg
		p.ImportPath = importpath
		return nil
	}

	// if this is the stdlib, then search vendor first.
	// this isn't real vendor support, just enough to make net/http compile.
	if i.Root == runtime.GOROOT() {
		path := pathpkg.Join("vendor", path)
		dir := filepath.Join(i.Root, "src", filepath.FromSlash(path))
		fi, err := os.Stat(dir)
		if err == nil && fi.IsDir() {
			err := loadPackage(path, dir)
			return p, err
		}
	}

	dir := filepath.Join(i.Root, "src", filepath.FromSlash(path))
	fi, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, errors.Errorf("import %q: not a directory", path)
	}
	err = loadPackage(path, dir)
	return p, err
}
