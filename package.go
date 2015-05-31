package gb

import "go/build"

import "fmt"

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
		if Stdlib[i] {
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
