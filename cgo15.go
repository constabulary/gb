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
	_, _, _, cgoLDFLAGS := cflags(pkg, true)
	_, pcLDFLAGS, err := pkgconfig(pkg)
	if err != nil {
		return fn(ErrTarget{err})
	}
	cgoLDFLAGS = append(cgoLDFLAGS, pcLDFLAGS...)
	if err := runcgo1(pkg, nil, cgoLDFLAGS); err != nil {
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
	for _, f := range cfiles {
		ofile := stripext(f) + ".o"
		ofiles = append(ofiles, ofile)
		targets = append(targets, rungcc1(pkg.Context, pkg.Dir, ofile, f))
	}

	for _, t := range targets {
		if err := t.Result(); err != nil {
			return fn(ErrTarget{err})
		}
	}

	ofile := filepath.Join(filepath.Dir(ofiles[0]), "_cgo_.o")
	if err := rungcc2(pkg, pkg.Dir, ofile, ofiles); err != nil {
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
	var cgoenv []string
	if len(ldflags) > 0 {
		flags := make([]string, len(ldflags))
		for i, f := range ldflags {
			flags[i] = strconv.Quote(f)
		}
		cgoenv = []string{"CGO_LDFLAGS=" + strings.Join(flags, " ")}
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
