package gb

import (
	"fmt"
	"path/filepath"

	"github.com/constabulary/gb/log"
)

// Resolver resolves packages.
type Resolver interface {
	// Srcdirs returns []string{ "$PROJECT/src", "$PROJECT/vendor/src" }
	Srcdirs() []string

	// ResolvePackage resolves the import path to a *Package
	ResolvePackage(path string) (*Package, error)
}

// ResolvePackages resolves import paths to packages.
func ResolvePackages(r Resolver, paths ...string) ([]*Package, error) {
	var pkgs []*Package
	for _, path := range paths {
		if path == "." {
			return nil, fmt.Errorf("%q is not a package", r.Srcdirs()[0])
		}
		path = relImportPath(r.Srcdirs()[0], path)
		pkg, err := r.ResolvePackage(path)
		if err != nil {
			return pkgs, fmt.Errorf("failed to resolve import path %q: %v", path, err)
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs, nil
}

func relImportPath(root, path string) string {
	if isRel(path) {
		var err error
		path, err = filepath.Rel(root, path)
		if err != nil {
			log.Fatalf("could not convert relative path %q to absolute: %v", path, err)
		}
	}
	return path
}

// isRel returns if an import path is relative or absolute.
func isRel(path string) bool {
	// TODO(dfc) should this be strings.StartsWith(".")
	return path == "."
}
