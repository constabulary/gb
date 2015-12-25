package gb

import (
	"fmt"
	"go/build"
	"path/filepath"
	"strings"
)

// Package represents a resolved package from the Project with respect to the Context.
type Package struct {
	*Context
	*build.Package
	Scope         string // scope: build, test, etc
	ExtraIncludes string // hook for test
	Stale         bool   // is the package out of date wrt. its cached copy
	Standard      bool   // is this package part of the standard library
}

// NewPackage creates a resolved Package.
func NewPackage(ctx *Context, p *build.Package) *Package {
	pkg := Package{
		Context: ctx,
		Package: p,
	}
	pkg.Stale = isStale(&pkg)
	pkg.Standard = pkg.Goroot // TODO(dfc) check this isn't set anywhere else
	return &pkg
}

// isMain returns true if this is a command, not being built in test scope, and
// not the testmain itself.
func (p *Package) isMain() bool {
	switch p.Scope {
	case "test":
		return strings.HasSuffix(p.ImportPath, "testmain")
	default:
		return p.Name == "main"
	}
}

// Imports returns the Pacakges that this Package depends on.
func (p *Package) Imports() []*Package {
	pkgs := make([]*Package, 0, len(p.Package.Imports))
	for _, i := range p.Package.Imports {
		pkg, ok := p.pkgs[i]
		if !ok {
			panic("could not locate package: " + i)
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs
}

func (p *Package) String() string {
	return fmt.Sprintf("%v", struct {
		Name, ImportPath, Dir string
	}{
		p.Name, p.ImportPath, p.Dir,
	})
}

// Complete indicates if this is a pure Go package
func (p *Package) Complete() bool {
	// If we're giving the compiler the entire package (no C etc files), tell it that,
	// so that it can give good error messages about forward declarations.
	// Exceptions: a few standard packages have forward declarations for
	// pieces supplied behind-the-scenes by package runtime.
	extFiles := len(p.CgoFiles) + len(p.CFiles) + len(p.CXXFiles) + len(p.MFiles) + len(p.SFiles) + len(p.SysoFiles) + len(p.SwigFiles) + len(p.SwigCXXFiles)
	if p.Standard {
		switch p.ImportPath {
		case "bytes", "net", "os", "runtime/pprof", "sync", "time":
			extFiles++
		}
	}
	return extFiles == 0
}

// Binfile returns the destination of the compiled target of this command.
func (pkg *Package) Binfile() string {
	// TODO(dfc) should have a check for package main, or should be merged in to objfile.
	var target string
	switch pkg.Scope {
	case "test":
		target = filepath.Join(pkg.Workdir(), filepath.FromSlash(pkg.ImportPath), "_test", binname(pkg))
	default:
		target = filepath.Join(pkg.Bindir(), binname(pkg))
	}

	// if this is a cross compile or there are build tags, add ctxString.
	if pkg.isCrossCompile() {
		target += "-" + pkg.ctxString()
	} else if len(pkg.buildtags) > 0 {
		target += "-" + strings.Join(pkg.buildtags, "-")
	}

	if pkg.gotargetos == "windows" {
		target += ".exe"
	}
	return target
}
