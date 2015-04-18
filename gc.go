package gb

// gc toolchain

import (
	"go/build"
	"os"
	"path/filepath"
)

type gcToolchain struct {
	goroot, goos, goarch string
	gc, cc, ld, as, pack string
}

func NewGcToolchain(goroot, goos, goarch string) (Toolchain, error) {
	tooldir := filepath.Join(goroot, "pkg", "tool", goos+"_"+goarch)
	archchar, err := build.ArchChar(goarch)
	if err != nil {
		return nil, err
	}
	return &gcToolchain{
		goroot: goroot,
		goos:   goos,
		goarch: goarch,
		gc:     filepath.Join(tooldir, archchar+"g"),
		ld:     filepath.Join(tooldir, archchar+"l"),
		as:     filepath.Join(tooldir, archchar+"a"),
		pack:   filepath.Join(tooldir, "pack"),
	}, nil
}

func (t *gcToolchain) Gc(searchpaths []string, importpath, srcdir, outfile string, files []string, complete bool) error {
	Debugf("gc:gc %v %v %v %v", importpath, srcdir, outfile, files)

	args := []string{"-p", importpath}
	for _, d := range searchpaths {
		args = append(args, "-I", d)
	}
	if complete {
		args = append(args, "-pack", "-complete")
	}
	args = append(args, "-o", outfile)
	args = append(args, files...)
	err := os.MkdirAll(filepath.Dir(outfile), 0755)
	if err != nil {
		return err
	}
	return run(srcdir, t.gc, args...)
}

func (t *gcToolchain) Cc(srcdir, objdir, outfile, cfile string) error {
	args := []string{"-F", "-V", "-w", "-I", objdir, "-I", filepath.Join(t.goroot, "pkg", t.goos+"_"+t.goarch)}
	args = append(args, "-o", outfile)
	args = append(args, cfile)
	return run(srcdir, t.cc, args...)
}

func (t *gcToolchain) Pack(afile string, ofiles ...string) error {
	args := []string{"grc", afile}
	args = append(args, ofiles...)
	dir := filepath.Dir(afile)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}
	return run(dir, t.pack, args...)
}

func (t *gcToolchain) Asm(srcdir, ofile, sfile string) error {
	args := []string{"-o", ofile, "-D", "GOOS_" + t.goos, "-D", "GOARCH_" + t.goarch, sfile}
	err := os.MkdirAll(filepath.Dir(ofile), 0755)
	if err != nil {
		return err
	}
	return run(srcdir, t.as, args...)
}

func (t *gcToolchain) Ld(searchpaths []string, outfile, afile string) error {
	args := []string{"-o", outfile}
	for _, d := range searchpaths {
		args = append(args, "-L", d)
	}
	args = append(args, afile)
	err := os.MkdirAll(filepath.Dir(outfile), 0755)
	if err != nil {
		return err
	}
	return run(".", t.ld, args...)
}
