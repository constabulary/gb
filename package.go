package gb

import (
	"fmt"
	"go/build"
	"path/filepath"
	"strings"
)

// Package represents a resolved package from the Project with respect to the Context.
type Package struct {
	*Context
	*build.Package
	Scope         string // scope: build, test, etc
	ExtraIncludes string // hook for test
	Stale         bool   // is the package out of date wrt. its cached copy
}

// NewPackage creates a resolved Package.
func NewPackage(ctx *Context, p *build.Package) *Package {
	pkg := Package{
		Context: ctx,
		Package: p,
	}
	// seed pkg.c so calling result never blocks
	pkg.Stale = isStale(&pkg)
	return &pkg
}

// isMain returns true if this is a command, a main package.
func (p *Package) isMain() bool {
	return p.Name == "main"
}

// Imports returns the Pacakges that this Package depends on.
func (p *Package) Imports() []*Package {
	pkgs := make([]*Package, 0, len(p.Package.Imports))
	for _, i := range p.Package.Imports {
		if shouldignore(i) {
			continue
		}
		pkg, ok := p.pkgs[i]
		if !ok {
			panic("could not locate package: " + i)
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs
}

func (p *Package) String() string {
	return fmt.Sprintf("%v", struct {
		Name, ImportPath, Dir string
	}{
		p.Name, p.ImportPath, p.Dir,
	})
}

// Complete indicates if this is a pure Go package
// TODO(dfc) this should be pure go with respect to tags and scope
func (p *Package) Complete() bool {
	has := func(s []string) bool { return len(s) > 0 }
	return !(has(p.SFiles) || has(p.CgoFiles))
}

// Objdir returns the destination for object files compiled for this Package.
func (pkg *Package) Objdir() string {
	switch pkg.Scope {
	case "test":
		ip := strings.TrimSuffix(filepath.FromSlash(pkg.ImportPath), "_test")
		return filepath.Join(pkg.Workdir(), ip, "_test", filepath.Dir(filepath.FromSlash(pkg.ImportPath)))
	default:
		return filepath.Join(pkg.Workdir(), filepath.Dir(filepath.FromSlash(pkg.ImportPath)))
	}
}

// loadPackage recursively resolves path and its imports and if successful
// stores those packages in the Context's internal package cache.
func loadPackage(c *Context, stack []string, path string) (*Package, error) {
	if build.IsLocalImport(path) {
		// sanity check
		return nil, fmt.Errorf("%q is not a valid import path", path)
	}
	if pkg, ok := c.pkgs[path]; ok {
		// already loaded, just return
		return pkg, nil
	}

	push := func(path string) {
		stack = append(stack, path)
	}
	pop := func(path string) {
		stack = stack[:len(stack)-1]
	}
	onStack := func(path string) bool {
		for _, p := range stack {
			if p == path {
				return true
			}
		}
		return false
	}

	p, err := c.Context.Import(path, c.Projectdir(), 0)
	if err != nil {
		return nil, err
	}
	push(path)
	var stale bool
	for _, i := range p.Imports {
		if shouldignore(i) {
			continue
		}
		if onStack(i) {
			push(i)
			return nil, fmt.Errorf("import cycle detected: %s", strings.Join(stack, " -> "))
		}
		pkg, err := loadPackage(c, stack, i)
		if err != nil {
			return nil, err
		}
		stale = stale || pkg.Stale
	}
	pop(path)

	pkg := Package{
		Context: c,
		Package: p,
	}
	pkg.Stale = stale || isStale(&pkg)
	c.pkgs[path] = &pkg
	return &pkg, nil
}
