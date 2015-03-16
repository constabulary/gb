package gb

import "go/build"

// Package represents a resolved package from the Project with respect to the Context.
type Package struct {
	target
	ctx *Context
	p   *build.Package
}

// resolvePackage resolves the package at path using the current context.
func resolvePackage(ctx *Context, path string) *Package {
	pkg := Package{
		ctx: ctx,
	}
	pkg.target = newTarget(pkg.findPackage(path))
	return &pkg
}

func (p *Package) findPackage(path string) func() error {
	ctx := p.ctx
	return func() error {
		var err error
		Debugf("Package::findPackage %v", path)
		p.p, err = ctx.Context.Import(path, ctx.Srcdir(), 0)
		return err
	}
}

// Name returns this package's name.
func (p *Package) Name() string {
	return p.p.Name
}
