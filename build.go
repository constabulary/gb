package gb

import (
	"fmt"
	"path"
	"path/filepath"
	"time"
)

// Build returns a Target representing the result of compiling the Package pkg
// and its dependencies. If pkg is a command, then the results of build include
// linking the final binary into pkg.Context.Bindir().
func Build(pkg *Package) Target {
	t := buildPackage(pkg)
	if err := t.Result(); err == nil {
		if pkg.isMain() {
			t = Ld(pkg, t.(PkgTarget))
		}
	}
	return t
}

// buildPackage returns a Target repesenting the results of compiling
// pkg and its dependencies.
func buildPackage(pkg *Package) Target {
	return pkg.ctx.targetOrMissing(fmt.Sprintf("compile:%s:%s", pkg.Scope, pkg.ImportPath), func() Target {
		deps := buildDependencies(pkg)
		return Compile(pkg, deps...)
	})
}

// Compile returns a Target representing all the steps required to build a go package.
func Compile(pkg *Package, deps ...Target) PkgTarget {
	return pkg.ctx.addTargetIfMissing(fmt.Sprintf("compile:%s:%s", pkg.Scope, pkg.ImportPath), func() Target {
		if !isStale(pkg) {
			return cachedPackage(pkg)
		}
		var gofiles []string
		gofiles = append(gofiles, pkg.GoFiles...)
		var objs []ObjTarget
		if len(pkg.CgoFiles) > 0 {
			// cgo, cgofiles := cgo(pkg, deps...)
			// deps = append(deps, cgo[0])
			// objs = append(objs, cgo...)
			// gofiles = append(gofiles, cgofiles...)
		}
		objs = append(objs, Gc(pkg, gofiles, deps...))
		for _, sfile := range pkg.SFiles {
			objs = append(objs, Asm(pkg, sfile))
		}
		if pkg.Complete() {
			return Install(pkg, objs[0].(PkgTarget))
		}
		return Install(pkg, Pack(pkg, objs...))
	}).(PkgTarget)
}

// ObjTarget represents a compiled Go object (.5, .6, etc)
type ObjTarget interface {
	Target

	// Objfile is the name of the file that is produced if the target is successful.
	Objfile() string
}

type gc struct {
	target
	pkg     *Package
	gofiles []string
}

func (g *gc) String() string {
	return fmt.Sprintf("compile %v", g.pkg)
}

func (g *gc) compile() error {
	t0 := time.Now()
	Infof("compile %v %v", g.pkg.ImportPath, g.gofiles)
	includes := g.pkg.ctx.IncludePaths()
	importpath := g.pkg.ImportPath
	if g.pkg.Scope == "test" && g.pkg.ExtraIncludes != "" {
		// TODO(dfc) gross
		includes = append([]string{g.pkg.ExtraIncludes}, includes...)
	}
	err := g.pkg.ctx.tc.Gc(includes, importpath, g.pkg.Dir, g.Objfile(), g.gofiles, g.pkg.Complete())
	g.pkg.ctx.Record("compile", time.Since(t0))
	return err
}

func (g *gc) Objfile() string {
	return objfile(g.pkg)
}

func objfile(pkg *Package) string {
	return filepath.Join(objdir(pkg), path.Base(pkg.ImportPath)+".a")
}

func (g *gc) Pkgfile() string {
	return g.Objfile()
}

// Gc returns a Target representing the result of compiling a set of gofiles with the Context specified gc Compiler.
func Gc(pkg *Package, gofiles []string, deps ...Target) interface {
	ObjTarget
	Pkgfile() string // implements PkgTarget
} {
	gc := gc{
		pkg:     pkg,
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
	c   chan error
	pkg *Package
}

func (p *pack) Result() error {
	err := <-p.c
	p.c <- err
	return err
}

func (p *pack) pack(objs ...ObjTarget) {
	Infof("pack [%v]", objs)
	afiles := make([]string, 0, len(objs))
	for _, obj := range objs {
		err := obj.Result()
		if err != nil {
			p.c <- err
			return
		}
		// pkg.a (compiled Go code) is always first
		afiles = append(afiles, obj.Objfile())
	}
	t0 := time.Now()
	err := p.pkg.ctx.tc.Pack(afiles...)
	p.pkg.ctx.Record("pack", time.Since(t0))
	p.c <- err
}

func (p *pack) Pkgfile() string {
	return objfile(p.pkg)
}

// Pack returns a Target representing the result of packing a
// set of Context specific object files into an archive.
func Pack(pkg *Package, deps ...ObjTarget) PkgTarget {
	pack := pack{
		c:   make(chan error, 1),
		pkg: pkg,
	}
	go pack.pack(deps...)
	return &pack
}

type asm struct {
	target
	pkg   *Package
	sfile string
}

func (a *asm) Objfile() string {
	return filepath.Join(a.pkg.ctx.workdir, a.pkg.ImportPath, stripext(a.sfile)+".6")
}

func (a *asm) asm() error {
	t0 := time.Now()
	Infof("asm %v", a.sfile)
	err := a.pkg.ctx.tc.Asm(a.pkg.Dir, a.Objfile(), filepath.Join(a.pkg.Dir, a.sfile))
	a.pkg.ctx.Record("asm", time.Since(t0))
	return err
}

// Asm returns a Target representing the result of assembling
// sfile with the Context specified asssembler.
func Asm(pkg *Package, sfile string) ObjTarget {
	asm := asm{
		pkg:   pkg,
		sfile: sfile,
	}
	asm.target = newTarget(asm.asm)
	return &asm
}

type ld struct {
	target
	pkg   *Package
	afile PkgTarget
}

func (l *ld) link() error {
	t0 := time.Now()
	target := filepath.Join(objdir(l.pkg), l.pkg.Name)
	Infof("link %v [%v]", target, l.afile.Pkgfile())
	includes := l.pkg.ctx.IncludePaths()
	if l.pkg.Scope == "test" && l.pkg.ExtraIncludes != "" {
		// TODO(dfc) gross
		includes = append([]string{l.pkg.ExtraIncludes}, includes...)
		target += ".test"
	}
	err := l.pkg.ctx.tc.Ld(includes, target, l.afile.Pkgfile())
	l.pkg.ctx.Record("link", time.Since(t0))
	return err
}

// Ld returns a Target representing the result of linking a
// Package into a command with the Context provided linker.
func Ld(pkg *Package, afile PkgTarget) Target {
	ld := ld{
		pkg:   pkg,
		afile: afile,
	}
	ld.target = newTarget(ld.link, afile)
	return &ld
}

func stripext(path string) string {
	ext := filepath.Ext(path)
	return path[:len(ext)]
}

// objdir returns the destination for object files compiled for this Package.
func objdir(pkg *Package) string {
	switch pkg.Scope {
	case "test":
		return filepath.Join(testobjdir(pkg), filepath.Dir(filepath.FromSlash(pkg.ImportPath)))
	default:
		return filepath.Join(pkg.ctx.workdir, filepath.Dir(filepath.FromSlash(pkg.ImportPath)))
	}
}

func testobjdir(pkg *Package) string {
	return filepath.Join(pkg.ctx.workdir, filepath.FromSlash(pkg.ImportPath), "_test")
}

// buildDependencies resolves the dependencies the package paths.
func buildDependencies(pkg *Package) []Target {
	var deps []Target
	for _, pkg := range pkg.Imports() {
		deps = append(deps, buildPackage(pkg))
	}
	return deps
}
