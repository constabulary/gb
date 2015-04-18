package gb

import (
	"fmt"
	"path/filepath"
)

// Install stores a copy of the compiled file in Project.Pkgdir
func Install(pkg *Package, t PkgTarget) PkgTarget {
	if pkg.ctx.SkipInstall {
		return t
	}
	if pkg.Scope == "test" {
		Debugf("%v is test scoped, not caching", pkg)
		return t
	}
	cc := cache{
		PkgTarget: t,
		dest:      filepath.Join(pkgdir(pkg), pkg.Name()+".a"),
	}
	cc.target = newTarget(cc.cache, t)
	return &cc
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
	return filepath.Join(pkgdir(c.pkg), c.pkg.Name()+".a")
}

func (c *cachedPkgTarget) String() string {
	return fmt.Sprintf("cached %v", c.pkg)
}

func (c *cachedPkgTarget) Result() error {
	// TODO(dfc) _, err := os.Stat(c.Pkgfile())
	return nil
}

type cache struct {
	target
	PkgTarget
	dest string
}

func (c *cache) String() string {
	return fmt.Sprintf("cache %v", c.PkgTarget)
}

func (c *cache) cache() error {
	Infof("cache %v", c.PkgTarget)
	return copyfile(c.dest, c.Pkgfile())
}

func (c *cache) Result() error {
	return c.target.Result()
}

// pkgdir returns the destination for object cached for this Package.
func pkgdir(pkg *Package) string {
	if pkg.Scope == "test" {
		panic("pkgdir called with test scope")
	}
	return filepath.Join(pkg.ctx.Pkgdir(), filepath.Dir(filepath.FromSlash(pkg.ImportPath)))
}

// isStale returns true if the source pkg is considered to be stale with
// respect to its cached copy.
func isStale(pkg *Package) bool {
	if pkg.ctx.Force {
		return true
	}
	return false
}
