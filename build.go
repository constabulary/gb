package gb

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Build builds each of pkgs in succession. If pkg is a command, then the results of build include
// linking the final binary into pkg.Context.Bindir().
func Build(pkgs ...*Package) error {
	targets := make(map[string]PkgTarget)
	roots := make([]Target, 0, len(pkgs))
	for _, pkg := range pkgs {
		target := buildPackage(targets, pkg)
		if pkg.isMain() {
			target = Ld(pkg, target.(PkgTarget))
		}
		roots = append(roots, target)
	}
	for _, root := range roots {
		if err := root.Result(); err != nil {
			return err
		}
	}
	return nil
}

// buildPackage returns a Target repesenting the results of compiling
// pkg and its dependencies.
func buildPackage(targets map[string]PkgTarget, pkg *Package) Target {
	if target, ok := targets[pkg.ImportPath]; ok {
		// already compiled
		return target
	}
	Debugf("buildPackage: %v", pkg.ImportPath)

	deps := BuildDependencies(targets, pkg)
	target := Compile(pkg, deps...)
	targets[pkg.ImportPath] = target
	return target
}

// BuildDependencies returns a []Target representing the results of
// compiling the dependencies of pkg.
func BuildDependencies(targets map[string]PkgTarget, pkg *Package) []Target {
	var deps []Target
	for _, i := range pkg.Imports() {
		deps = append(deps, buildPackage(targets, i))
	}
	return deps
}

// Compile returns a Target representing all the steps required to build a go package.
func Compile(pkg *Package, deps ...Target) PkgTarget {
	if !pkg.Stale {
		return cachedPackage(pkg)
	}
	var gofiles []string
	gofiles = append(gofiles, pkg.GoFiles...)
	var cgoobj []ObjTarget
	if len(pkg.CgoFiles) > 0 {
		var cgofiles []string
		cgoobj, cgofiles = cgo(pkg)
		for _, o := range cgoobj {
			deps = append(deps, o)
		}
		gofiles = append(gofiles, cgofiles...)
	}
	objs := []ObjTarget{Gc(pkg, gofiles, deps...)}
	if len(cgoobj) > 0 {
		objs = append(objs, cgoobj...)
	}
	for _, sfile := range pkg.SFiles {
		objs = append(objs, Asm(pkg, sfile))
	}
	if pkg.Complete() {
		return Install(pkg, objs[0].(PkgTarget))
	}
	return Install(pkg, Pack(pkg, objs...))
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
	if g.pkg.Scope != "test" {
		// only log compilation message if not in test scope
		Infof(g.pkg.ImportPath)
	}
	includes := g.pkg.IncludePaths()
	importpath := g.pkg.ImportPath
	if g.pkg.Scope == "test" && g.pkg.ExtraIncludes != "" {
		// TODO(dfc) gross
		includes = append([]string{g.pkg.ExtraIncludes}, includes...)
	}
	for i := range g.gofiles {
		if filepath.IsAbs(g.gofiles[i]) {
			// terrible hack for cgo files which come with an absolute path
			continue
		}
		g.gofiles[i], _ = filepath.Rel(g.pkg.Projectdir(), filepath.Join(g.pkg.Dir, g.gofiles[i]))
	}
	err := g.pkg.tc.Gc(g.pkg, includes, importpath, g.pkg.Projectdir(), g.Objfile(), g.gofiles, g.pkg.Complete())
	g.pkg.Record("compile", time.Since(t0))
	return err
}

func (g *gc) Objfile() string {
	return objfile(g.pkg)
}

func (g *gc) Pkgfile() string {
	return g.Objfile()
}

type objpkgtarget interface {
	ObjTarget
	Pkgfile() string // implements PkgTarget
}

// Gc returns a Target representing the result of compiling a set of gofiles with the Context specified gc Compiler.
func Gc(pkg *Package, gofiles []string, deps ...Target) objpkgtarget {
	if len(gofiles) == 0 {
		return ErrTarget{fmt.Errorf("Gc: no Gofiles provided")}
	}
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
	err := p.pkg.tc.Pack(p.pkg, afiles...)
	p.pkg.Record("pack", time.Since(t0))
	p.c <- err
}

func (p *pack) Pkgfile() string {
	return objfile(p.pkg)
}

// Pack returns a Target representing the result of packing a
// set of Context specific object files into an archive.
func Pack(pkg *Package, deps ...ObjTarget) PkgTarget {
	if len(deps) < 2 {
		return ErrTarget{fmt.Errorf("Pack requires at least two arguments: %v", deps)}
	}
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

func (a *asm) String() string {
	return fmt.Sprintf("asm %v", a.sfile)
}

func (a *asm) Objfile() string {
	return filepath.Join(a.pkg.Workdir(), a.pkg.ImportPath, stripext(a.sfile)+".6")
}

func (a *asm) asm() error {
	t0 := time.Now()
	err := a.pkg.tc.Asm(a.pkg, a.pkg.Dir, a.Objfile(), filepath.Join(a.pkg.Dir, a.sfile))
	a.pkg.Record("asm", time.Since(t0))
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
	target := l.pkg.Binfile()
	if err := mkdir(filepath.Dir(target)); err != nil {
		return err
	}

	includes := l.pkg.IncludePaths()
	if l.pkg.Scope == "test" && l.pkg.ExtraIncludes != "" {
		// TODO(dfc) gross
		includes = append([]string{l.pkg.ExtraIncludes}, includes...)
		target += ".test"
	}
	err := l.pkg.tc.Ld(l.pkg, includes, l.pkg.ldflags, target, l.afile.Pkgfile())
	l.pkg.Record("link", time.Since(t0))
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

// objfile returns the name of the object file for this package
func objfile(pkg *Package) string {
	return filepath.Join(pkg.Objdir(), objname(pkg))
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

func objname(pkg *Package) string {
	switch pkg.Name {
	case "main":
		return filepath.Join(filepath.Base(filepath.FromSlash(pkg.ImportPath)), "main.a")
	default:
		return filepath.Base(filepath.FromSlash(pkg.ImportPath)) + ".a"
	}
}

// Binfile returns the destination of the compiled target of this command.
// TODO(dfc) this should be Target.
func (pkg *Package) Binfile() string {
	// TODO(dfc) should have a check for package main, or should be merged in to objfile.
	var target string
	switch pkg.Scope {
	case "test":
		target = filepath.Join(pkg.Workdir(), filepath.FromSlash(pkg.ImportPath), "_test", binname(pkg))
	default:
		target = filepath.Join(pkg.Bindir(), binname(pkg))
	}
	if pkg.GOOS == "windows" {
		target += ".exe"
	}
	return target
}

func binname(pkg *Package) string {
	switch {
	case pkg.Name == "main":
		return filepath.Base(filepath.FromSlash(pkg.ImportPath))
	case pkg.Scope == "test":
		return pkg.Name
	default:
		panic("binname called with non main package: " + pkg.ImportPath)
	}
}
