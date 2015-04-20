package gb

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"
)

// Install stores a copy of the compiled file in Project.Pkgdir
func Install(pkg *Package, t PkgTarget) PkgTarget {
	if pkg.ctx.SkipInstall {
		return t
	}
	if pkg.isMain() {
		Debugf("%v is a main package, not installing", pkg)
		return t
	}
	if pkg.Scope == "test" {
		Debugf("%v is test scoped, not installing", pkg)
		return t
	}
	i := install{
		PkgTarget: t,
		dest:      pkgfile(pkg),
	}
	i.target = newTarget(i.install, t)
	return &i
}

// cachePackage returns a PkgTarget representing the cached output of
// pkg.
func cachedPackage(pkg *Package) *cachedPkgTarget {
	return &cachedPkgTarget{
		pkg: pkg,
	}
}

type cachedPkgTarget struct {
	pkg *Package
}

func (c *cachedPkgTarget) Pkgfile() string {
	return pkgfile(c.pkg)
}

func (c *cachedPkgTarget) String() string {
	return fmt.Sprintf("cached %v", c.pkg)
}

func (c *cachedPkgTarget) Result() error {
	// TODO(dfc) _, err := os.Stat(c.Pkgfile())
	return nil
}

type install struct {
	target
	PkgTarget
	dest string
}

func (i *install) String() string {
	return fmt.Sprintf("cache %v", i.PkgTarget)
}

func (i *install) install() error {
	Infof("install %v", i.PkgTarget)
	return copyfile(i.dest, i.Pkgfile())
}

func (i *install) Result() error {
	return i.target.Result()
}

// pkgdir returns the destination for object cached for this Package.
func pkgdir(pkg *Package) string {
	if pkg.Scope == "test" {
		panic("pkgdir called with test scope")
	}
	return filepath.Join(pkg.ctx.Pkgdir(), filepath.Dir(filepath.FromSlash(pkg.p.ImportPath)))
}

func pkgfile(pkg *Package) string {
	return filepath.Join(pkgdir(pkg), path.Base(pkg.p.ImportPath)+".a")
}

// isStale returns true if the source pkg is considered to be stale with
// respect to its installed version.
func isStale(pkg *Package) bool {
	if pkg.ctx.Force {
		return true
	}

	if pkg.Scope == "test" {
		// tests are always stale, they are never installed
		return true
	}

	// Package is stale if completely unbuilt.
	var built time.Time
	if fi, err := os.Stat(pkgfile(pkg)); err == nil {
		built = fi.ModTime()
	}
	if built.IsZero() {
		return true
	}

	olderThan := func(file string) bool {
		fi, err := os.Stat(file)
		return err != nil || fi.ModTime().After(built)
	}

	p := pkg.p

	srcs := stringList(p.GoFiles, p.CFiles, p.CXXFiles, p.MFiles, p.HFiles, p.SFiles, p.CgoFiles, p.SysoFiles, p.SwigFiles, p.SwigCXXFiles)
	for _, src := range srcs {
		if olderThan(filepath.Join(p.Dir, src)) {
			return true
		}
	}

	return false
}

// stale returns true if pkg or any of its transitive dependencies are not stale
func stale(pkg *Package) bool {
	var f func(*Package) bool
	f = func(pkg *Package) bool {
		if err := pkg.Result(); err != nil {
			return true
		}	
		for _, dep := range pkg.p.Imports {
			if f(resolvePackage(pkg.ctx, dep)) {
				return true
			}
		}
		return isStale(pkg)
	}
	return f(pkg)
}

func stringList(args ...[]string) []string {
	var l []string
	for _, arg := range args {
		l = append(l, arg...)
	}
	return l
}
