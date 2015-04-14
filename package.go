package gb

import "go/build"

// Package represents a resolved package from the Project with respect to the Context.
type Package struct {
	c          chan error
	ctx        *Context
	p          *build.Package
	ImportPath string
}

// resolvePackage resolves the package at path using the current context.
func resolvePackage(ctx *Context, path string) *Package {
	return ctx.addTargetIfMissing("package:"+path, func() Target {
		pkg := Package{
			c:          make(chan error, 1),
			ctx:        ctx,
			ImportPath: path,
		}
		go pkg.resolvePackage(path)
		return &pkg
	}).(*Package)
}

// newPackage creates a resolved Package.
func newPackage(ctx *Context, p *build.Package) *Package {
	pkg := Package{
		c:          make(chan error, 1),
		ctx:        ctx,
		ImportPath: p.ImportPath,
		p:          p,
	}
	return &pkg
}

// Name returns this package's name.
func (p *Package) Name() string {
	return p.p.Name
}

func (p *Package) Result() error {
	err := <-p.c
	p.c <- err
	return err
}

func (p *Package) resolvePackage(path string) {
	Debugf("Package::findPackage %v", path)
	pkg, err := p.ctx.Context.Import(path, p.ctx.Projectdir(), 0)
	p.p = pkg
	p.c <- err
}
