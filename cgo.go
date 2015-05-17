package gb

import (
	"path/filepath"
)

// cgo support functions

// cgo returns a slice of post processed source files and a slice of
// ObjTargets representing the result of compilation of the post .c
// output.
func cgo(pkg *Package) ([]ObjTarget, []string) {
	err := runcgo(pkg)
	if err != nil {
		return []ObjTarget{errTarget{err}}, nil
	}

	return nil, nil
}

// runcgo invokes the cgo tool installed in the current ctx.goroot
func runcgo(pkg *Package) error {
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

func cgotool(ctx *Context) string {
	return filepath.Join(ctx.GOROOT, "pkg", "tool", ctx.GOOS+"_"+ctx.GOARCH, "cgo")
}
