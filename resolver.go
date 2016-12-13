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

func nullImporter() func(string) (*build.Package, error) {
	return func(path string) (*build.Package, error) {
		return nil, errors.Errorf("import %q not found", path)
	}
}

type importerFn func(string) (*build.Package, error)

func (fn importerFn) Import(path string) (*build.Package, error) {
	return fn(path)
}

func srcImporter(parent Importer, child func(string) (*build.Package, error)) func(string) (*build.Package, error) {
	return func(path string) (*build.Package, error) {
		pkg, err := child(path)
		if err == nil {
			return pkg, nil
		}

		// gb expects, when there is a failure to resolve packages that
		// live in $PROJECT/src that the importer for that directory
		// will report them.

		pkg, err2 := parent.Import(path)
		if err2 == nil {
			return pkg, nil
		}
		return nil, err
	}
}

func childFirstImporter(parent, child func(string) (*build.Package, error)) func(string) (*build.Package, error) {
	return func(path string) (*build.Package, error) {
		pkg, err := child(path)
		if err != nil {
			return parent(path)
		}
		return pkg, nil
	}
}

func fixupImporter(importer func(string) (*build.Package, error)) func(string) (*build.Package, error) {
	return func(path string) (*build.Package, error) {
		pkg, err := importer(path)
		switch err.(type) {
		case *os.PathError:
			return nil, errors.Wrapf(err, "import %q: not found", path)
		default:
			return pkg, err
		}
	}
}

type importErr struct {
	path string
	msg  string
}

func (e *importErr) Error() string {
	return fmt.Sprintf("import %q: %v", e.path, e.msg)
}

func dirImporter(ctx *build.Context, dir string) func(string) (*build.Package, error) {
	return func(path string) (*build.Package, error) {
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
			pkg, err := ctx.ImportDir(dir, 0)
			if err != nil {
				return err
			}
			p = pkg
			p.ImportPath = importpath
			return nil
		}

		// if this is the stdlib, then search vendor first.
		// this isn't real vendor support, just enough to make net/http compile.
		if dir == runtime.GOROOT() {
			path := pathpkg.Join("vendor", path)
			dir := filepath.Join(dir, "src", filepath.FromSlash(path))
			fi, err := os.Stat(dir)
			if err == nil && fi.IsDir() {
				err := loadPackage(path, dir)
				return p, err
			}
		}

		dir := filepath.Join(dir, "src", filepath.FromSlash(path))
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
}
