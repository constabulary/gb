package gb

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Package represents a resolved package from the Project with respect to the Context.
type Package struct {
	*Context
	*build.Package
	TestScope bool
	NotStale  bool // this package _and_ all its dependencies are not stale
	Main      bool // is this a command
	Imports   []*Package
}

// newPackage creates a resolved Package without setting pkg.Stale.
func (ctx *Context) newPackage(p *build.Package) (*Package, error) {
	pkg := &Package{
		Context: ctx,
		Package: p,
	}
	for _, i := range p.Imports {
		dep, ok := ctx.pkgs[i]
		if !ok {
			return nil, errors.Errorf("newPackage(%q): could not locate dependant package %q ", p.Name, i)
		}
		pkg.Imports = append(pkg.Imports, dep)
	}
	return pkg, nil
}

func (p *Package) String() string {
	return fmt.Sprintf("%s {Name:%s, Dir:%s}", p.ImportPath, p.Name, p.Dir)
}

func (p *Package) includePaths() []string {
	includes := p.Context.includePaths()
	switch {
	case p.TestScope && p.Main:
		ip := filepath.Dir(filepath.FromSlash(p.ImportPath))
		return append([]string{filepath.Join(p.Context.Workdir(), ip, "_test")}, includes...)
	case p.TestScope:
		ip := strings.TrimSuffix(filepath.FromSlash(p.ImportPath), "_test")
		return append([]string{filepath.Join(p.Context.Workdir(), ip, "_test")}, includes...)
	default:
		return includes
	}
}

// complete indicates if this is a pure Go package
func (p *Package) complete() bool {
	// If we're giving the compiler the entire package (no C etc files), tell it that,
	// so that it can give good error messages about forward declarations.
	// Exceptions: a few standard packages have forward declarations for
	// pieces supplied behind-the-scenes by package runtime.
	extFiles := len(p.CgoFiles) + len(p.CFiles) + len(p.CXXFiles) + len(p.MFiles) + len(p.SFiles) + len(p.SysoFiles) + len(p.SwigFiles) + len(p.SwigCXXFiles)
	if p.Goroot {
		switch p.ImportPath {
		case "bytes", "internal/poll", "net", "os", "runtime/pprof", "sync", "syscall", "time":
			extFiles++
		}
	}
	return extFiles == 0
}

// Binfile returns the destination of the compiled target of this command.
func (pkg *Package) Binfile() string {
	target := filepath.Join(pkg.bindir(), pkg.binname())

	// if this is a cross compile or GOOS/GOARCH are both defined or there are build tags, add ctxString.
	if pkg.isCrossCompile() || (os.Getenv("GOOS") != "" && os.Getenv("GOARCH") != "") {
		target += "-" + pkg.ctxString()
	} else if len(pkg.buildtags) > 0 {
		target += "-" + strings.Join(pkg.buildtags, "-")
	}

	if pkg.gotargetos == "windows" {
		target += ".exe"
	}
	return target
}

func (pkg *Package) bindir() string {
	switch {
	case pkg.TestScope:
		return filepath.Join(pkg.Context.Workdir(), filepath.FromSlash(pkg.ImportPath), "_test")
	default:
		return pkg.Context.bindir()
	}
}

func (pkg *Package) Workdir() string {
	path := filepath.FromSlash(pkg.ImportPath)
	dir := filepath.Dir(path)
	switch {
	case pkg.TestScope:
		ip := strings.TrimSuffix(path, "_test")
		return filepath.Join(pkg.Context.Workdir(), ip, "_test", dir)
	default:
		return filepath.Join(pkg.Context.Workdir(), dir)
	}
}

// objfile returns the name of the object file for this package
func (pkg *Package) objfile() string {
	return filepath.Join(pkg.Workdir(), pkg.objname())
}

func (pkg *Package) objname() string {
	return pkg.pkgname() + ".a"
}

func (pkg *Package) pkgname() string {
	// TODO(dfc) use pkg path instead?
	return filepath.Base(filepath.FromSlash(pkg.ImportPath))
}

func (pkg *Package) binname() string {
	if !pkg.Main {
		panic("binname called with non main package: " + pkg.ImportPath)
	}
	// TODO(dfc) use pkg path instead?
	return filepath.Base(filepath.FromSlash(pkg.ImportPath))
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
func (pkg *Package) installpath() string {
	if pkg.TestScope {
		panic("installpath called with test scope")
	}
	return filepath.Join(pkg.Pkgdir(), filepath.FromSlash(pkg.ImportPath)+".a")
}

// pkgpath returns the destination for object cached for this Package.
func (pkg *Package) pkgpath() string {
	importpath := filepath.FromSlash(pkg.ImportPath) + ".a"
	switch {
	case pkg.isCrossCompile():
		return filepath.Join(pkg.Pkgdir(), importpath)
	case pkg.Goroot && pkg.race:
		// race enabled standard lib
		return filepath.Join(runtime.GOROOT(), "pkg", pkg.gotargetos+"_"+pkg.gotargetarch+"_race", importpath)
	case pkg.Goroot:
		// standard lib
		return filepath.Join(runtime.GOROOT(), "pkg", pkg.gotargetos+"_"+pkg.gotargetarch, importpath)
	default:
		return filepath.Join(pkg.Pkgdir(), importpath)
	}
}

// isStale returns true if the source pkg is considered to be stale with
// respect to its installed version.
func (pkg *Package) isStale() bool {
	switch pkg.ImportPath {
	case "C", "unsafe":
		// synthetic packages are never stale
		return false
	}

	if !pkg.Goroot && pkg.Force {
		return true
	}

	// tests are always stale, they are never installed
	if pkg.TestScope {
		return true
	}

	// Package is stale if completely unbuilt.
	var built time.Time
	if fi, err := os.Stat(pkg.pkgpath()); err == nil {
		built = fi.ModTime()
	}

	if built.IsZero() {
		pkg.debug("%s is missing", pkg.pkgpath())
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
	if !pkg.Goroot {
		if olderThan(pkg.tc.compiler()) {
			pkg.debug("%s is older than %s", pkg.pkgpath(), pkg.tc.compiler())
			return true
		}
		if pkg.Main && olderThan(pkg.tc.linker()) {
			pkg.debug("%s is older than %s", pkg.pkgpath(), pkg.tc.compiler())
			return true
		}
	}

	if pkg.Goroot && !pkg.isCrossCompile() {
		// if this is a standard lib package, and we are not cross compiling
		// then assume the package is up to date. This also works around
		// golang/go#13769.
		return false
	}

	// Package is stale if a dependency is newer.
	for _, p := range pkg.Imports {
		if p.ImportPath == "C" || p.ImportPath == "unsafe" {
			continue // ignore stale imports of synthetic packages
		}
		if olderThan(p.pkgpath()) {
			pkg.debug("%s is older than %s", pkg.pkgpath(), p.pkgpath())
			return true
		}
	}

	// if the main package is up to date but _newer_ than the binary (which
	// could have been removed), then consider it stale.
	if pkg.Main && newerThan(pkg.Binfile()) {
		pkg.debug("%s is newer than %s", pkg.pkgpath(), pkg.Binfile())
		return true
	}

	srcs := stringList(pkg.GoFiles, pkg.CFiles, pkg.CXXFiles, pkg.MFiles, pkg.HFiles, pkg.SFiles, pkg.CgoFiles, pkg.SysoFiles, pkg.SwigFiles, pkg.SwigCXXFiles)

	for _, src := range srcs {
		if olderThan(filepath.Join(pkg.Dir, src)) {
			pkg.debug("%s is older than %s", pkg.pkgpath(), filepath.Join(pkg.Dir, src))
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
