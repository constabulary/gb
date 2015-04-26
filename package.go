package gb

import "go/build"
import "fmt"

// Package represents a resolved package from the Project with respect to the Context.
type Package struct {
	ctx *Context
	*build.Package
	Scope         string // scope: build, test, etc
	ExtraIncludes string // hook for test
	Stale         bool   // is the package out of date wrt. its cached copy
}

// newPackage creates a resolved Package.
func newPackage(ctx *Context, p *build.Package) *Package {
	pkg := Package{
		ctx:     ctx,
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
		if stdlib[i] {
			continue
		}
		pkg, ok := p.ctx.pkgs[i]
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
