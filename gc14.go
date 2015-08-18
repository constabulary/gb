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
	envor := func(key, def string) string {
		if v := os.Getenv(key); v != "" {
			return v
		} else {
			return def
		}
	}

	defaults := []func(*gcoption){
		func(opt *gcoption) {
			opt.goos = envor("GOOS", runtime.GOOS)
		},
		func(opt *gcoption) {
			opt.goarch = envor("GOARCH", runtime.GOARCH)
		},
	}

	var options gcoption
	for _, opt := range append(defaults, opts...) {
		opt(&options)
	}

	return func(c *Context) error {
		goroot := runtime.GOROOT()
		goos := options.goos
		goarch := options.goarch

		// cross-compliation is not supported yet #31
		if goos != runtime.GOOS || goarch != runtime.GOARCH {
			return fmt.Errorf("cross compilation from host %s/%s to target %s/%s not supported. See issue #31", runtime.GOOS, runtime.GOARCH, goos, goarch)
		}

		archchar, err := build.ArchChar(goarch)
		if err != nil {
			return err
		}
		tooldir := filepath.Join(goroot, "pkg", "tool", goos+"_"+goarch)
		c.tc = &gcToolchain{
			goos:   goos,
			goarch: goarch,
			gc:     filepath.Join(tooldir, archchar+"g"),
			ld:     filepath.Join(tooldir, archchar+"l"),
			as:     filepath.Join(tooldir, archchar+"a"),
			cc:     filepath.Join(tooldir, archchar+"c"),
			pack:   filepath.Join(tooldir, "pack"),
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
	includedir := filepath.Join(runtime.GOROOT(), "pkg", t.goos+"_"+t.goarch)
	args := []string{"-o", ofile, "-D", "GOOS_" + t.goos, "-D", "GOARCH_" + t.goarch, "-I", includedir, sfile}
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
		"-I", filepath.Join(pkg.GOROOT, "pkg", pkg.GOOS+"_"+pkg.GOARCH), // for runtime.h
		"-o", ofile,
		"-D", "GOOS_" + pkg.GOOS,
		"-D", "GOARCH_" + pkg.GOARCH,
		cfile,
	}
	return run(pkg.Dir, nil, t.cc, args...)
}
