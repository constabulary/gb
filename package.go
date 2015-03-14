package gb

import "go/build"

// Package represents a resolved package from the Project with respect to the Context.
type Package struct {
	target
	p *build.Package
}

// ResolvePackage resolves the package at path using the current context.
func ResolvePackage(ctx *Context, path string) *Package {
	var p Package
	p.target = newTarget(p.findPackage(ctx, path))
	return &p
}

func (p *Package) findPackage(ctx *Context, path string) func() error {
	return func() error {
		var err error
		p.p, err = ctx.Context.Import(path, ctx.Srcdir(), 0)
		return err
	}
}

// Name returns this package's name.
func (p *Package) Name() string {
	return p.p.Name
}
