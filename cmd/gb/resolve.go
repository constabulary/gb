package main

import (
	"fmt"

	"github.com/constabulary/gb"
)

// Resolver resolves packages.
type Resolver interface {
	// ResolvePackage resolves the import path to a *Package
	ResolvePackage(path string) (*gb.Package, error)
}

// ResolvePackages resolves import paths to packages.
func resolvePackages(r Resolver, paths ...string) ([]*gb.Package, error) {
	var pkgs []*gb.Package
	for _, path := range paths {
		pkg, err := r.ResolvePackage(path)
		if err != nil {
			return pkgs, fmt.Errorf("failed to resolve import path %q: %v", path, err)
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs, nil
}
