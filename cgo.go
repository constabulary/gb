package gb

import (
	"path/filepath"
)

// cgo support functions

// cgo returns a slice of post processed source files and a slice of
// ObjTargets representing the result of compilation of the post .c
// output.
func cgo(pkg *Package) ([]ObjTarget, []string) {
	if err := runcgo1(pkg); err != nil {
		return []ObjTarget{errTarget{err}}, nil
	}

	var cgofiles []string
	for _, f := range pkg.CgoFiles {
		cgofiles = append(cgofiles, filepath.Join(objdir(pkg), f+".cgo1.go"))
	}
	cfiles := []string{
		filepath.Join(objdir(pkg), "_cgo_main.c"),
		filepath.Join(objdir(pkg), "_cgo_export.c"),
	}
	cfiles = append(cfiles, pkg.CFiles...)

	for _, f := range pkg.CgoFiles {
		cfiles = append(cfiles, filepath.Join(objdir(pkg), f+".cgo2.c"))
	}

	var ofiles []string
	for _, f := range cfiles {
		ext := filepath.Ext(f)
		ofile := f[:len(f)-len(ext)] + ".o"
		ofiles = append(ofiles, ofile)
		if err := rungcc1(pkg.Dir, ofile, f); err != nil {
			return []ObjTarget{errTarget{err}}, nil
		}
	}

	return nil, cgofiles
}

// runcgo1 invokes the cgo tool to process pkg.CgoFiles.
func runcgo1(pkg *Package) error {
	cgo := cgotool(pkg.ctx)
	objdir := objdir(pkg)
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
	return run(pkg.Dir, cgo, args...)
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

func cgotool(ctx *Context) string {
	return filepath.Join(ctx.GOROOT, "pkg", "tool", ctx.GOOS+"_"+ctx.GOARCH, "cgo")
}
