package gb

import "fmt"

// Build returns a Target representing the result of compiling the Package pkg
// and its dependencies. If pkg is a command, then the results of build include
// linking the final binary into pkg.Context.Bindir().
func Build(pkg *Package) Target {
	if err := pkg.Result(); err != nil {
		// TODO(dfc)
		panic(err)
	}
	if pkg.Name() == "main" {
		return buildCommand(pkg)
	}
	return buildPackage(pkg)
}

// buildPackage returns a Target repesenting the results of compiling
// pkg and its dependencies.
func buildPackage(pkg *Package) Target {
	var deps []Target
	for _, dep := range pkg.p.Imports {
		// TODO(dfc) use project.Spec
		pkg := resolvePackage(pkg.ctx, dep)
		deps = append(deps, buildPackage(pkg))
	}
	return pkg.ctx.addTargetIfMissing(pkg, func() Target { return Compile(pkg.ctx, pkg, deps...) })
}

// buildCommand returns a Target repesenting the results of compiling
// pkg as a command and linking the result into pkg.Context.Bindir().
func buildCommand(pkg *Package) Target {
	var deps []Target
	for _, dep := range pkg.p.Imports {
		pkg := resolvePackage(pkg.ctx, dep)
		deps = append(deps, buildPackage(pkg))
	}
	compile := Compile(pkg.ctx, pkg, deps...)
	ld := Ld(pkg.ctx, pkg, compile.(PkgTarget))
	return ld
}

type errTarget struct {
	error
}

func (e errTarget) Result() error {
	return e.error
}

// Compile returns a Target representing all the steps required to build a go package.
func Compile(ctx *Context, pkg *Package, deps ...Target) Target {
	if err := pkg.Result(); err != nil {
		return errTarget{err}
	}
	var gofiles []string
	gofiles = append(gofiles, pkg.p.GoFiles...)
	var objs []Target
	if len(pkg.p.CgoFiles) > 0 {
		/**cgo, cgofiles := cgo(ctx, pkg, deps)
		  deps = append(deps, cgo[0])
		  objs = append(objs, cgo...)
		  gofiles = append(gofiles, cgofiles...) */
	}
	objs = append(objs, Gc(ctx, pkg, gofiles, deps...))
	for _, sfile := range pkg.p.SFiles {
		objs = append(objs, Asm(ctx, pkg, sfile))
	}
	return Pack(ctx, pkg, objs...)
}

// ObjTarget represents a compiled Go object (.5, .6, etc)
type ObjTarget interface {
	Target

	// Objfile is the name of the file that is produced if the target is successful.
	Objfile() string
}

type gc struct {
	target
	ctx     *Context
	pkg     *Package
	gofiles []string
	objfile string
}

func (g *gc) compile() error {
	Debugf("gc::compile %v (%v)", g.pkg.Name(), g.gofiles)
	return nil
}

func (g *gc) Objfile() string { return g.objfile }

// Gc returns a Target representing the result of compiling a set of gofiles with the Context specified gc Compiler.
func Gc(ctx *Context, pkg *Package, gofiles []string, deps ...Target) ObjTarget {
	gc := gc{
		ctx:     ctx,
		gofiles: gofiles,
	}
	gc.target = newTarget(gc.compile, deps...)
	return &gc
}

// PkgTarget represents a Target that produces a pkg (.a) file.
type PkgTarget interface {
	Target

	// Pkgfile returns the name of the file that is produced by the Target if successful.
	Pkgfile() string
}

type pack struct {
	target
	ctx   *Context
	deps  []Target
	afile string
}

func (p *pack) pack() error {
	var ofiles []string
	for _, dep := range p.deps {
		switch dep := dep.(type) {
		case ObjTarget:
			ofiles = append(ofiles, dep.Objfile())
		default:
			return fmt.Errorf("unexpected Target %T", dep)
		}
	}
	return nil
}

func (p *pack) Pkgfile() string { return p.afile }

// Pack returns a Target representing the result of packing a
// set of Context specific object files into an archive.
func Pack(ctx *Context, pkg *Package, deps ...Target) PkgTarget {
	pack := pack{
		ctx:  ctx,
		deps: deps,
	}
	pack.target = newTarget(pack.pack, deps...)
	return &pack
}

type asm struct {
	target
	ctx   *Context
	ofile string
}

func (a *asm) Objfile() string { return a.ofile }

func (a *asm) asm() error {
	return nil
}

// Asm returns a Target representing the result of assembling
// sfile with the Context specified asssembler.
func Asm(ctx *Context, pkg *Package, sfile string) ObjTarget {
	asm := asm{
		ctx: ctx,
	}
	asm.target = newTarget(asm.asm)
	return &asm
}

type ld struct {
	target
	ctx   *Context
	afile PkgTarget
}

func (l *ld) link() error {
	return nil
}

// Ld returns a Target representing the result of linking a
// Package into a command with the Context provided linker.
func Ld(ctx *Context, pkg *Package, afile PkgTarget) Target {
	ld := ld{
		ctx:   ctx,
		afile: afile,
	}
	ld.target = newTarget(ld.link, afile)
	return &ld
}
