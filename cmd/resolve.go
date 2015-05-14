package cmd

import (
	"fmt"
	"go/build"
	"path/filepath"

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

// ResolvePackages resolves args, specified as import paths to packages.
func ResolvePackages(r Resolver, projectroot string, args ...string) ([]*gb.Package, error) {
	var pkgs []*gb.Package
	for _, arg := range args {
		if arg == "." {
			var err error
			arg, err = filepath.Rel(r.Srcdirs()[0], projectroot)
			if err != nil {
				return pkgs, err
			}
		}
		pkg, err := r.ResolvePackage(arg)
		if err != nil {
			if _, ok := err.(*build.NoGoError); ok {
				gb.Debugf("skipping %q", arg)
				continue
			}
			return pkgs, fmt.Errorf("failed to resolve package %q: %v", arg, err)
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs, nil
}

// ResolvePackagesWithTests is similar to ResolvePackages however
// it also loads the test and external test packages of args into
// the context.
func ResolvePackagesWithTests(r Resolver, projectroot string, args ...string) ([]*gb.Package, error) {
	var pkgs []*gb.Package
	for _, arg := range args {
		if arg == "." {
			var err error
			arg, err = filepath.Rel(r.Srcdirs()[0], projectroot)
			if err != nil {
				return pkgs, err
			}
		}
		pkg, err := r.ResolvePackageWithTests(arg)
		if err != nil {
			if _, ok := err.(*build.NoGoError); ok {
				gb.Debugf("skipping %q", arg)
				continue
			}
			return pkgs, fmt.Errorf("failed to resolve package %q: %v", arg, err)
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs, nil
}
