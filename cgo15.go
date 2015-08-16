// +build go1.5

package gb

import (
	"path/filepath"
	"strings"
)

// cgo support functions

// cgo produces a an Action representing the cgo steps
// an ofile representing the result of the cgo steps
// a set of .go files for compilation, and an error.
func cgo(pkg *Package) (*Action, []string, []string, error) {

	// collect cflags and ldflags from the package
	// the environment, and pkg-config.
	_, cgoCFLAGS, _, cgoLDFLAGS := cflags(pkg, false)
	pcCFLAGS, pcLDFLAGS, err := pkgconfig(pkg)
	if err != nil {
		return nil, nil, nil, err
	}
	cgoCFLAGS = append(cgoCFLAGS, pcCFLAGS...)
	cgoLDFLAGS = append(cgoLDFLAGS, pcLDFLAGS...)

	runcgo1 := []*Action{
		&Action{
			Name: "runcgo1: " + pkg.ImportPath,
			Task: TaskFn(func() error {
				return runcgo1(pkg, cgoCFLAGS, cgoLDFLAGS)
			}),
		},
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
	var gcc1 []*Action
	for _, cfile := range cfiles {
		cfile := cfile
		ofile := filepath.Join(pkg.Objdir(), stripext(filepath.Base(cfile))+".o")
		ofiles = append(ofiles, ofile)
		gcc1 = append(gcc1, &Action{
			Name: "rungcc1: " + pkg.ImportPath + ": " + cfile,
			Deps: runcgo1,
			Task: TaskFn(func() error {
				return rungcc1(pkg, cgoCFLAGS, ofile, cfile)
			}),
		})
	}

	ofile := filepath.Join(filepath.Dir(ofiles[0]), "_cgo_.o")
	gcc2 := Action{
		Name: "rungcc2: " + pkg.ImportPath + ": _cgo_.o",
		Deps: gcc1,
		Task: TaskFn(func() error {
			return rungcc2(pkg, cgoCFLAGS, cgoLDFLAGS, ofile, ofiles)
		}),
	}

	dynout := filepath.Join(pkg.Objdir(), "_cgo_import.go")
	runcgo2 := Action{
		Name: "runcgo2: " + pkg.ImportPath,
		Deps: []*Action{&gcc2},
		Task: TaskFn(func() error {
			return runcgo2(pkg, dynout, ofile)
		}),
	}
	cgofiles = append(cgofiles, dynout)

	allo := filepath.Join(filepath.Dir(ofiles[0]), "_all.o")
	action := Action{
		Name: "rungcc3: " + pkg.ImportPath,
		Deps: []*Action{&runcgo2},
		Task: TaskFn(func() error {
			return rungcc3(pkg.Context, pkg.Dir, allo, ofiles[1:]) // skip _cgo_main.o
		}),
	}

	return &action, []string{allo}, cgofiles, nil
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
	args = append(args, cflags...)
	args = append(args, ldflags...)
	args = append(args, pkg.CgoFiles...)

	cgoenv := []string{
		"CGO_CFLAGS=" + strings.Join(quoteFlags(cflags), " "),
		"CGO_LDFLAGS=" + strings.Join(quoteFlags(ldflags), " "),
	}
	return pkg.run(pkg.Dir, cgoenv, cgo, args...)
}

// runcgo2 invokes the cgo tool to create _cgo_import.go
func runcgo2(pkg *Package, dynout, ofile string) error {
	cgo := cgotool(pkg.Context)
	objdir := pkg.Objdir()

	args := []string{
		"-objdir", objdir,
		"-dynpackage", pkg.Name,
		"-dynimport", ofile,
		"-dynout", dynout,
	}
	return pkg.run(pkg.Dir, nil, cgo, args...)
}
