package cmd

import (
	"fmt"
	"go/build"

	"github.com/constabulary/gb"
)

// Resolver resolves packages.
type Resolver interface {
	// Srcdirs returns []string{ "$PROJECT/src", "$PROJECT/vendor/src" }
	Srcdirs() []string

	// ResolvePackage resolves the import path to a *gb.Package
	ResolvePackage(path string) (*gb.Package, error)

	// ResolvePackagesWithTests is similar to ResolvePackages however
	// it also loads the test and external test packages of args into
	// the context.
	ResolvePackageWithTests(path string) (*gb.Package, error)
}

// ResolvePackages resolves import paths to packages.
func ResolvePackages(r Resolver, paths ...string) ([]*gb.Package, error) {
	var pkgs []*gb.Package
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

// ResolvePackagesWithTests is similar to ResolvePackages however
// it also loads the test and external test packages of args into
// the context.
func ResolvePackagesWithTests(r Resolver, paths ...string) ([]*gb.Package, error) {
	var pkgs []*gb.Package
	for _, path := range paths {
		path = relImportPath(r.Srcdirs()[0], path)
		pkg, err := r.ResolvePackageWithTests(path)
		if err != nil {
			if _, ok := err.(*build.NoGoError); ok {
				gb.Debugf("skipping %q", path)
				continue
			}
			return pkgs, fmt.Errorf("failed to resolve package %q: %v", path, err)
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs, nil
}
