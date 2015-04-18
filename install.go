package gb

import (
	"fmt"
	"os"
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
	return filepath.Join(pkg.ctx.Pkgdir(), filepath.Dir(filepath.FromSlash(pkg.ImportPath)))
}

func pkgfile(pkg *Package) string {
	return filepath.Join(pkgdir(pkg), pkg.Name()+".a")
}

// isStale returns true if the source pkg is considered to be stale with
// respect to its cached copy.
func isStale(pkg *Package) bool {
	if pkg.ctx.Force {
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

	return false
}
