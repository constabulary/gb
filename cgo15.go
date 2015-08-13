// +build go1.5

package gb

import (
	"path/filepath"
	"strconv"
	"strings"
)

// cgo support functions

// cgo returns a slice of post processed source files and an
// ObjTargets representing the result of compilation of the post .c
// output.
func cgo(pkg *Package) ([]ObjTarget, []string) {
	fn := func(t ...ObjTarget) ([]ObjTarget, []string) {
		return t, nil
	}
	_, _, cgoCFLAGS, cgoLDFLAGS := cflags(pkg, true)
	pcCFLAGS, pcLDFLAGS, err := pkgconfig(pkg)
	if err != nil {
		return fn(ErrTarget{err})
	}
	cgoCFLAGS = append(cgoCFLAGS, pcCFLAGS...)
	cgoLDFLAGS = append(cgoLDFLAGS, pcLDFLAGS...)
	if err := runcgo1(pkg, cgoCFLAGS, cgoLDFLAGS); err != nil {
		return fn(ErrTarget{err})
	}

	cgofiles := []string{filepath.Join(pkg.Objdir(), "_cgo_gotypes.go")}
	for _, f := range pkg.CgoFiles {
		cgofiles = append(cgofiles, filepath.Join(pkg.Objdir(), stripext(f)+".cgo1.go"))
	}
	cfiles := []string{
		filepath.Join(pkg.Objdir(), "_cgo_main.c"),
		filepath.Join(pkg.Objdir(), "_cgo_export.c"),
	}
	cfiles = append(cfiles, pkg.CFiles...)

	for _, f := range pkg.CgoFiles {
		cfiles = append(cfiles, filepath.Join(pkg.Objdir(), stripext(f)+".cgo2.c"))
	}

	var ofiles []string
	var targets []Target
	for _, cfile := range cfiles {
		ofile := filepath.Join(pkg.Objdir(), stripext(filepath.Base(cfile))+".o")
		ofiles = append(ofiles, ofile)
		targets = append(targets, rungcc1(pkg, ofile, cfile))
	}

	for _, t := range targets {
		if err := t.Result(); err != nil {
			return fn(ErrTarget{err})
		}
	}

	ofile := filepath.Join(filepath.Dir(ofiles[0]), "_cgo_.o")
	if err := rungcc2(pkg, cgoCFLAGS, cgoLDFLAGS, ofile, ofiles); err != nil {
		return fn(ErrTarget{err})
	}

	dynout, err := runcgo2(pkg, ofile)
	if err != nil {
		return fn(ErrTarget{err})
	}
	cgofiles = append(cgofiles, dynout)

	allo, err := rungcc3(pkg.Context, pkg.Dir, ofiles[1:]) // skip _cgo_main.o
	if err != nil {
		return fn(ErrTarget{err})
	}

	return []ObjTarget{cgoTarget(allo)}, cgofiles
}

// runcgo1 invokes the cgo tool to process pkg.CgoFiles.
func runcgo1(pkg *Package, cflags, ldflags []string) error {
	cgo := cgotool(pkg.Context)
	objdir := pkg.Objdir()
	if err := mkdir(objdir); err != nil {
		return err
	}

	args := []string{
		"-objdir", objdir,
		"-importpath", pkg.ImportPath,
		"--",
		"-I", objdir,
		"-I", pkg.Dir,
	}
	args = append(args, pkg.CgoFiles...)

	// Update $CGO_LDFLAGS with p.CgoLDFLAGS.
	cgoenv := []string{
		"CGO_CFLAGS=" + strings.Join(quoteFlags(cflags), " "),
		"CGO_LDFLAGS=" + strings.Join(quoteFlags(ldflags), " "),
	}
	return pkg.run(pkg.Dir, cgoenv, cgo, args...)
}

// runcgo2 invokes the cgo tool to create _cgo_import.go
func runcgo2(pkg *Package, ofile string) (string, error) {
	cgo := cgotool(pkg.Context)
	objdir := pkg.Objdir()
	dynout := filepath.Join(objdir, "_cgo_import.go")

	args := []string{
		"-objdir", objdir,
		"-dynpackage", pkg.Name,
		"-dynimport", ofile,
		"-dynout", dynout,
	}
	return dynout, pkg.run(pkg.Dir, nil, cgo, args...)
}
