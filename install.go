package gb

import (
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/constabulary/gb/debug"
)

// pkgpath returns the destination for object cached for this Package.
func pkgpath(pkg *Package) string {
	importpath := filepath.FromSlash(pkg.ImportPath) + ".a"
	switch {
	case pkg.isCrossCompile():
		return filepath.Join(pkg.Pkgdir(), importpath)
	case pkg.Standard && pkg.race:
		// race enabled standard lib
		return filepath.Join(runtime.GOROOT(), "pkg", pkg.gotargetos+"_"+pkg.gotargetarch+"_race", importpath)
	case pkg.Standard:
		// standard lib
		return filepath.Join(runtime.GOROOT(), "pkg", pkg.gotargetos+"_"+pkg.gotargetarch, importpath)
	default:
		return filepath.Join(pkg.Pkgdir(), importpath)
	}
}

// installpath returns the distination to cache this package's compiled .a file.
// pkgpath and installpath differ in that the former returns the location where you will find
// a previously cached .a file, the latter returns the location where an installed file
// will be placed.
//
// The difference is subtle. pkgpath must deal with the possibility that the file is from the
// standard library and is previously compiled. installpath will always return a path for the
// project's pkg/ directory in the case that the stdlib is out of date, or not compiled for
// a specific architecture.
func installpath(pkg *Package) string {
	if pkg.Scope == "test" {
		panic("installpath called with test scope")
	}
	return filepath.Join(pkg.Pkgdir(), filepath.FromSlash(pkg.ImportPath)+".a")
}

// isStale returns true if the source pkg is considered to be stale with
// respect to its installed version.
func isStale(pkg *Package) bool {
	switch pkg.ImportPath {
	case "C", "unsafe":
		// synthetic packages are never stale
		return false
	}

	if !pkg.Standard && pkg.Force {
		return true
	}

	// tests are always stale, they are never installed
	if pkg.Scope == "test" {
		return true
	}

	// Package is stale if completely unbuilt.
	var built time.Time
	if fi, err := os.Stat(pkgpath(pkg)); err == nil {
		built = fi.ModTime()
	}

	if built.IsZero() {
		debug.Debugf("%s is missing", pkgpath(pkg))
		return true
	}

	olderThan := func(file string) bool {
		fi, err := os.Stat(file)
		return err != nil || fi.ModTime().After(built)
	}

	newerThan := func(file string) bool {
		fi, err := os.Stat(file)
		return err != nil || fi.ModTime().Before(built)
	}

	// As a courtesy to developers installing new versions of the compiler
	// frequently, define that packages are stale if they are
	// older than the compiler, and commands if they are older than
	// the linker.  This heuristic will not work if the binaries are
	// back-dated, as some binary distributions may do, but it does handle
	// a very common case.
	if !pkg.Standard {
		if olderThan(pkg.tc.compiler()) {
			debug.Debugf("%s is older than %s", pkgpath(pkg), pkg.tc.compiler())
			return true
		}
		if pkg.IsCommand() && olderThan(pkg.tc.linker()) {
			debug.Debugf("%s is older than %s", pkgpath(pkg), pkg.tc.compiler())
			return true
		}
	}

	// Package is stale if a dependency is newer.
	for _, p := range pkg.Imports() {
		if p.ImportPath == "C" || p.ImportPath == "unsafe" {
			continue // ignore stale imports of synthetic packages
		}
		if olderThan(pkgpath(p)) {
			debug.Debugf("%s is older than %s", pkgpath(pkg), pkgpath(p))
			return true
		}
	}

	// if the main package is up to date but _newer_ than the binary (which
	// could have been removed), then consider it stale.
	if pkg.isMain() && newerThan(pkg.Binfile()) {
		debug.Debugf("%s is newer than %s", pkgpath(pkg), pkg.Binfile())
		return true
	}

	srcs := stringList(pkg.GoFiles, pkg.CFiles, pkg.CXXFiles, pkg.MFiles, pkg.HFiles, pkg.SFiles, pkg.CgoFiles, pkg.SysoFiles, pkg.SwigFiles, pkg.SwigCXXFiles)

	for _, src := range srcs {
		if olderThan(filepath.Join(pkg.Dir, src)) {
			debug.Debugf("%s is older than %s", pkgpath(pkg), filepath.Join(pkg.Dir, src))
			return true
		}
	}

	return false
}

func stringList(args ...[]string) []string {
	var l []string
	for _, arg := range args {
		l = append(l, arg...)
	}
	return l
}
