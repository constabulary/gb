package gb

// gc toolchain

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
)

type gcToolchain struct {
	goroot, goos, goarch string
	gc, cc, ld, as, pack string
}

func GcToolchain(goroot, goos, goarch string) func(c *Context) error {
	return func(c *Context) error {
		archchar, err := build.ArchChar(goarch)
		if err != nil {
			return err
		}
		tooldir := filepath.Join(goroot, "pkg", "tool", goos+"_"+goarch)
		c.tc = &gcToolchain{
			goroot: goroot,
			goos:   goos,
			goarch: goarch,
			gc:     filepath.Join(tooldir, archchar+"g"),
			ld:     filepath.Join(tooldir, archchar+"l"),
			as:     filepath.Join(tooldir, archchar+"a"),
			pack:   filepath.Join(tooldir, "pack"),
		}
		return nil
	}
}

func (t *gcToolchain) Gc(searchpaths []string, importpath, srcdir, outfile string, files []string, complete bool) error {
	Debugf("gc:gc %v", struct {
		ImportPath string
		Srcdir     string
		Outfile    string
		Gofiles    []string
	}{importpath, srcdir, outfile, files})

	args := []string{"-p", importpath, "-pack"}
	args = append(args, "-o", outfile)
	for _, d := range searchpaths {
		args = append(args, "-I", d)
	}
	if complete {
		args = append(args, "-complete")
	}
	args = append(args, files...)
	err := os.MkdirAll(filepath.Dir(outfile), 0755)
	if err != nil {
		return fmt.Errorf("gc:gc: %v", err)
	}
	return run(srcdir, t.gc, args...)
}

func (t *gcToolchain) Cc(srcdir, objdir, outfile, cfile string) error {
	args := []string{"-F", "-V", "-w", "-I", objdir, "-I", filepath.Join(t.goroot, "pkg", t.goos+"_"+t.goarch)}
	args = append(args, "-o", outfile)
	args = append(args, cfile)
	return run(srcdir, t.cc, args...)
}

func (t *gcToolchain) Pack(afiles ...string) error {
	args := []string{"r"}
	args = append(args, afiles...)
	dir := filepath.Dir(afiles[0])
	return run(dir, t.pack, args...)
}

func (t *gcToolchain) Asm(srcdir, ofile, sfile string) error {
	args := []string{"-o", ofile, "-D", "GOOS_" + t.goos, "-D", "GOARCH_" + t.goarch, sfile}
	err := os.MkdirAll(filepath.Dir(ofile), 0755)
	if err != nil {
		return fmt.Errorf("gc:asm: %v", err)
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
		return fmt.Errorf("gc:ld: %v", err)
	}
	return run(".", t.ld, args...)
}
