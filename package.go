package gb

import "go/build"
import "fmt"

// Package represents a resolved package from the Project with respect to the Context.
type Package struct {
	c             chan error
	ctx           *Context
	p             *build.Package
	Scope         string // scope: build, test, etc
	ExtraIncludes string // hook for test
	Stale		bool	// is the package out of date wrt. its cached copy
}

// resolvePackage resolves the package at path using the current context.
func resolvePackage(ctx *Context, path string) *Package {
	return ctx.addTargetIfMissing("package:"+path, func() Target {
		pkg := Package{
			c:   make(chan error, 1),
			ctx: ctx,
		}
		go pkg.resolvePackage(path)
		return &pkg
	}).(*Package)
}

// newPackage creates a resolved Package.
func newPackage(ctx *Context, p *build.Package) *Package {
	pkg := Package{
		c:   make(chan error, 1),
		ctx: ctx,
		p:   p,
	}
	// seed pkg.c so calling result never blocks
	pkg.Stale = isStale(&pkg)
	pkg.c <- nil
	return &pkg
}

// Name returns this package's name.
func (p *Package) Name() string {
	return p.p.Name
}

// isMain returns true if this is a command, a main package.
func (p *Package) isMain() bool {
	return p.p.Name == "main"
}

func (p *Package) String() string {
	return fmt.Sprintf("%v", struct {
		Name, ImportPath, Dir string
	}{
		p.Name(), p.p.ImportPath, p.p.Dir,
	})
}

func (p *Package) Result() error {
	err := <-p.c
	p.c <- err
	return err
}

func (p *Package) resolvePackage(path string) {
	Debugf("Package::findPackage %v", path)
	pkg, err := p.ctx.Context.Import(path, p.ctx.Projectdir(), 0)
	if err != nil {
		err = fmt.Errorf("resolvePackage(%q): %v", path, err)
		p.c <- err
		return 
	}
	p.p = pkg
	p.c <- err
	return
}

// Complete indicates if this is a pure Go package
// TODO(dfc) this should be pure go with respect to tags and scope
func (p *Package) Complete() bool {
	has := func(s []string) bool { return len(s) > 0 }
	return !(has(p.p.SFiles) || has(p.p.CgoFiles))
}
