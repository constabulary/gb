package gb

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/constabulary/gb/internal/importer"
	"github.com/pkg/errors"
)

// Package represents a resolved package from the Project with respect to the Context.
type Package struct {
	*Context
	*importer.Package
	TestScope     bool
	ExtraIncludes string // hook for test
	Stale         bool   // is the package out of date wrt. its cached copy
	Imports       []*Package
}

// newPackage creates a resolved Package without setting pkg.Stale.
func newPackage(ctx *Context, p *importer.Package) (*Package, error) {
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

// isMain returns true if this is a command, not being built in test scope, and
// not the testmain itself.
func (p *Package) isMain() bool {
	if p.TestScope {
		return strings.HasSuffix(p.ImportPath, "testmain")
	}
	return p.Name == "main"
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
	target := filepath.Join(pkg.Bindir(), binname(pkg))
	if pkg.TestScope {
		target = filepath.Join(pkg.Workdir(), filepath.FromSlash(pkg.ImportPath), "_test", binname(pkg))
	}

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
