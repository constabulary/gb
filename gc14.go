// +build !go1.5

package gb

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"runtime"
)

func GcToolchain(opts ...func(*gcoption)) func(c *Context) error {
	return func(c *Context) error {
		// TODO(dfc) this should come from the context, not the runtime.
		goroot := runtime.GOROOT()

		// cross-compliation is not supported yet #31
		if c.gohostos != c.gotargetos || c.gohostarch != c.gotargetarch {
			return fmt.Errorf("cross compilation from host %s/%s to target %s/%s not supported with Go 1.4", c.gohostos, c.gohostarch, c.gotargetos, c.gotargetarch)
		}

		archchar, err := build.ArchChar(c.gotargetarch)
		if err != nil {
			return err
		}
		tooldir := filepath.Join(goroot, "pkg", "tool", c.gohostos+"_"+c.gohostarch)
		c.tc = &gcToolchain{
			gc:   filepath.Join(tooldir, archchar+"g"),
			ld:   filepath.Join(tooldir, archchar+"l"),
			as:   filepath.Join(tooldir, archchar+"a"),
			cc:   filepath.Join(tooldir, archchar+"c"),
			pack: filepath.Join(tooldir, "pack"),
		}
		return nil
	}
}

func (t *gcToolchain) Gc(pkg *Package, searchpaths []string, importpath, srcdir, outfile string, files []string) error {
	Debugf("gc:gc %v %v %v %v", importpath, srcdir, outfile, files)

	args := append(pkg.gcflags, "-p", importpath, "-pack")
	args = append(args, "-o", outfile)
	for _, d := range searchpaths {
		args = append(args, "-I", d)
	}
	if pkg.Complete() {
		args = append(args, "-complete")
	}
	args = append(args, files...)
	if err := mkdir(filepath.Dir(outfile)); err != nil {
		return fmt.Errorf("gc:gc: %v", err)
	}
	return runOut(os.Stdout, srcdir, nil, t.gc, args...)
}

func (t *gcToolchain) Asm(pkg *Package, srcdir, ofile, sfile string) error {
	includedir := filepath.Join(pkg.Context.Context.GOROOT, "pkg", pkg.gotargetos+"_"+pkg.gotargetarch)
	args := []string{"-o", ofile, "-D", "GOOS_" + pkg.gotargetos, "-D", "GOARCH_" + pkg.gotargetarch, "-I", includedir, sfile}
	if err := mkdir(filepath.Dir(ofile)); err != nil {
		return fmt.Errorf("gc:asm: %v", err)
	}
	return run(srcdir, nil, t.as, args...)
}

func (t *gcToolchain) Ld(pkg *Package, searchpaths []string, outfile, afile string) error {
	args := append(pkg.ldflags, "-o", outfile)
	for _, d := range searchpaths {
		args = append(args, "-L", d)
	}
	args = append(args, "-extld="+gcc()) // TODO(dfc) go 1.5+, "-buildmode=exe")
	args = append(args, afile)
	if err := mkdir(filepath.Dir(outfile)); err != nil {
		return fmt.Errorf("gc:ld: %v", err)
	}
	return run(".", nil, t.ld, args...)
}

func (t *gcToolchain) Cc(pkg *Package, ofile, cfile string) error {
	args := []string{
		"-F", "-V", "-w",
		"-trimpath", pkg.Workdir(),
		"-I", pkg.Objdir(),
		"-I", filepath.Join(pkg.Context.Context.GOROOT, "pkg", pkg.gohostos+"_"+pkg.gohostarch), // for runtime.h
		"-o", ofile,
		"-D", "GOOS_" + pkg.gotargetos,
		"-D", "GOARCH_" + pkg.gotargetarch,
		cfile,
	}
	return run(pkg.Dir, nil, t.cc, args...)
}
