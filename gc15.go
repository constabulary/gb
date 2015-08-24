// +build go1.5

package gb

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// gc toolchain

func GcToolchain(opts ...func(*gcoption)) func(c *Context) error {
	envor := func(key, def string) string {
		if v, ok := os.LookupEnv(key); ok {
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
		gohostos := runtime.GOOS
		gohostarch := runtime.GOARCH
		gotargetos := options.goos
		gotargetarch := options.goarch

		// cross-compliation is not supported yet #31
		if gohostos != gotargetos || gohostarch != gotargetarch {
			return fmt.Errorf("cross compilation from host %s/%s to target %s/%s not supported. See issue #31", gohostos, gohostarch, gotargetos, gotargetarch)
		}

		tooldir := filepath.Join(goroot, "pkg", "tool", gohostos+"_"+gohostarch)
		c.tc = &gcToolchain{
			gohostos:     gohostos,
			gohostarch:   gohostarch,
			gotargetos:   gotargetos,
			gotargetarch: gotargetarch,
			gc:           filepath.Join(tooldir, "compile"),
			ld:           filepath.Join(tooldir, "link"),
			as:           filepath.Join(tooldir, "asm"),
			pack:         filepath.Join(tooldir, "pack"),
		}
		return nil
	}
}

func (t *gcToolchain) Gc(pkg *Package, searchpaths []string, importpath, srcdir, outfile string, files []string) error {
	args := append(pkg.gcflags, "-p", importpath, "-pack")
	args = append(args, "-o", outfile)
	for _, d := range searchpaths {
		args = append(args, "-I", d)
	}
	if pkg.Standard && pkg.ImportPath == "runtime" {
		// runtime compiles with a special gc flag to emit
		// additional reflect type data.
		args = append(args, "-+")
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
	includedir := filepath.Join(runtime.GOROOT(), "pkg", "include")
	args := []string{"-o", ofile, "-D", "GOOS_" + t.gotargetos, "-D", "GOARCH_" + t.gotargetos, "-I", includedir, sfile}
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
	args = append(args, "-extld="+gcc(), "-buildmode=exe")
	args = append(args, afile)
	if err := mkdir(filepath.Dir(outfile)); err != nil {
		return fmt.Errorf("gc:ld: %v", err)
	}
	return run(".", nil, t.ld, args...)
}

func (t *gcToolchain) Cc(pkg *Package, ofile, cfile string) error {
	return fmt.Errorf("gc15 does not support cc")
}
