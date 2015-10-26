package gb

import (
	"fmt"
	"path/filepath"
)

// Resolver resolves packages.
type Resolver interface {
	// ResolvePackage resolves the import path to a *Package
	ResolvePackage(path string) (*Package, error)
}

// ResolvePackages resolves import paths to packages.
func ResolvePackages(r Resolver, paths ...string) ([]*Package, error) {
	var pkgs []*Package
	for _, path := range paths {
		pkg, err := r.ResolvePackage(path)
		if err != nil {
			return pkgs, fmt.Errorf("failed to resolve import path %q: %v", path, err)
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs, nil
}

func relImportPath(root, path string) (string, error) {
	if isRel(path) {
		return filepath.Rel(root, path)
	}
	return path, nil
}

// isRel returns if an import path is relative or absolute.
func isRel(path string) bool {
	// TODO(dfc) should this be strings.StartsWith(".")
	return path == "."
}
