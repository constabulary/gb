// +build !go1.5

package gb

import (
	"bytes"
	"path/filepath"
	"go/build"
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
	if err := runcgo1(pkg); err != nil {
		return fn(ErrTarget{err})
	}

	defun, err := runcc1(pkg)
	if err != nil {
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
	for _, f := range cfiles {
		ofile := stripext(f) + ".o"
		ofiles = append(ofiles, ofile)
		if err := rungcc1(pkg.Dir, ofile, f); err != nil {
			return fn(ErrTarget{err})
		}
	}

	ofile, err := rungcc2(pkg.Dir, ofiles)
	if err != nil {
		return fn(ErrTarget{err})
	}

	_, err = runcgo2(pkg, ofile)
	if err != nil {
		return fn(ErrTarget{err})
	}
	imports, err := runcc2(pkg) // TODO(dfc) should compile dynout from above
	if err != nil {
		return fn(ErrTarget{err})
	}

        allo, err := rungcc3(pkg.Dir, ofiles[1:]) // skip _cgo_main.o
        if err != nil {
		return fn(ErrTarget{err})
        }

	return []ObjTarget{cgoTarget(defun), cgoTarget(imports), cgoTarget(allo)}, cgofiles
}

type cgoTarget string

func (t cgoTarget) Objfile() string { return string(t) }
func (t cgoTarget) Result() error   { return nil }

// runcgo1 invokes the cgo tool to process pkg.CgoFiles.
func runcgo1(pkg *Package) error {
	cgo := cgotool(pkg.Context)
	objdir := pkg.Objdir()
	if err := mkdir(objdir); err != nil {
		return err
	}

	args := []string{
		"-objdir", objdir,
		"--",
		"-I", pkg.Dir,
	}
	args = append(args, pkg.CgoFiles...)
	return run(pkg.Dir, cgo, args...)
}

// runcgo2 invokes the cgo tool to create _cgo_import.go
func runcgo2(pkg *Package, ofile string) (string, error) {
	cgo := cgotool(pkg.Context)
	objdir := pkg.Objdir()
	dynout := filepath.Join(objdir, "_cgo_import.c")

	args := []string{
		"-objdir", objdir,
		"-dynimport", ofile,
		"-dynout", dynout,
	}
	return dynout, run(pkg.Dir, cgo, args...)
}

func runcc1(pkg *Package) (string, error) {
	archchar, err := build.ArchChar(pkg.GOARCH)
	if err != nil {
		return "", err
	}
	cc := filepath.Join(pkg.GOROOT, "pkg", "tool", pkg.GOOS+"_"+pkg.GOARCH, archchar +"c")
	objdir := pkg.Objdir()
	ofile := filepath.Join(objdir, "_cgo_defun."+archchar)
	args := []string{
		"-F", "-V", "-w", 
		"-trimpath", pkg.Workdir(),
		"-I", objdir,
		"-I", filepath.Join(pkg.GOROOT, "pkg", pkg.GOOS+"_"+pkg.GOARCH), // for runtime.h
		"-o", ofile,
		"-D", "GOOS_"+pkg.GOOS,
		"-D", "GOARCH_"+pkg.GOARCH,
		filepath.Join(objdir, "_cgo_defun.c"),
	}
	return ofile, run(pkg.Dir, cc, args...)
}

// /home/dfc/go/pkg/tool/linux_amd64/6c -F -V -w -trimpath $WORK -I $WORK/cgomain/_obj/ -I /home/dfc/go/pkg/linux_amd64 -o $WORK/cgomain/_obj/_cgo_import.6 -D GOOS_linux -D GOARCH_amd64 $WORK/cgomain/_obj/_cgo_import.c
func runcc2(pkg *Package) (string, error) {
	archchar, err := build.ArchChar(pkg.GOARCH)
	if err != nil {
		return "", err
	}
	cc := filepath.Join(pkg.GOROOT, "pkg", "tool", pkg.GOOS+"_"+pkg.GOARCH, archchar +"c")
	objdir := pkg.Objdir()
	ofile := filepath.Join(objdir, "_cgo_import."+archchar)
	args := []string{
		"-F", "-V", "-w", 
		"-trimpath", pkg.Workdir(),
		"-I", objdir,
		"-o", ofile,
		"-D", "GOOS_"+pkg.GOOS,
		"-D", "GOARCH_"+pkg.GOARCH,
		filepath.Join(objdir, "_cgo_import.c"),
	}
	return ofile, run(pkg.Dir, cc, args...)
}

// rungcc1 invokes gcc to compile cfile into ofile
func rungcc1(dir, ofile, cfile string) error {
        gcc := "gcc" // TODO(dfc) handle $CC and clang
        args := []string{
                "-fPIC", "-m64", "-pthread", "-fmessage-length=0",
                "-I", dir,
                "-I", filepath.Dir(ofile),
                "-g", "-O2",
                "-o", ofile,
                "-c", cfile,
        }
        return run(dir, gcc, args...)
}

// rungcc2 links the o files from rungcc1 into a single _cgo_.o.
func rungcc2(dir string, ofiles []string) (string, error) {
	gcc := "gcc" // TODO(dfc) handle $CC and clang
	ofile := filepath.Join(filepath.Dir(ofiles[0]), "_cgo_.o")
	args := []string{
		"-fPIC", "-m64", "-pthread", "-fmessage-length=0",
		"-o", ofile,
	}
	args = append(args, ofiles...)
	args = append(args, "-g", "-O2") // this has to go at the end, because reasons!
	return ofile, run(dir, gcc, args...)
}

// rungcc3 links all previous ofiles together with libgcc into a single _all.o.
func rungcc3(dir string, ofiles []string) (string, error) {
	libgcc, err := libgcc()
	if err != nil {
		return "", nil
	}
	gcc := "gcc" // TODO(dfc) handle $CC and clang
	ofile := filepath.Join(filepath.Dir(ofiles[0]), "_all.o")
	args := []string{
		"-fPIC", "-m64", "-pthread", "-fmessage-length=0",
		"-g", "-O2",
		"-o", ofile,
	}
	args = append(args, ofiles...)
	args = append(args, "-Wl,-r", "-nostdlib", libgcc, "-Wl,--build-id=none")
	return ofile, run(dir, gcc, args...)
}

// libgcc returns the value of gcc -print-libgcc-file-name.
func libgcc() (string, error) {
	gcc := "gcc" // TODO(dfc) handle $CC and clang
	args := []string{
		"-fPIC", "-m64", "-pthread", "-fmessage-length=0",
		"-print-libgcc-file-name",
	}
	var buf bytes.Buffer
	err := runOut(&buf, ".", gcc, args...)
	return strings.TrimSpace(buf.String()), err
}

func cgotool(ctx *Context) string {
	return filepath.Join(ctx.GOROOT, "pkg", "tool", ctx.GOOS+"_"+ctx.GOARCH, "cgo")
}
