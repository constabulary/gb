// +build go1.5

package gb

import (
	"fmt"
	"path/filepath"
	"runtime"
)

// gc toolchain

func GcToolchain(opts ...func(*gcoption)) func(c *Context) error {
	defaults := []func(*gcoption){
		func(opt *gcoption) {
			opt.goos = runtime.GOOS
		},
		func(opt *gcoption) {
			opt.goarch = runtime.GOARCH
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
		tooldir := filepath.Join(goroot, "pkg", "tool", goos+"_"+goarch)
		c.tc = &gcToolchain{
			goos:   goos,
			goarch: goarch,
			gc:     filepath.Join(tooldir, "compile"),
			ld:     filepath.Join(tooldir, "link"),
			as:     filepath.Join(tooldir, "asm"),
			pack:   filepath.Join(tooldir, "pack"),
		}
		return nil
	}
}

func (t *gcToolchain) Gc(pkg *Package, searchpaths []string, importpath, srcdir, outfile string, files []string, complete bool) error {
	args := []string{"-p", importpath, "-pack"}
	args = append(args, "-o", outfile)
	for _, d := range searchpaths {
		args = append(args, "-I", d)
	}
	if complete {
		args = append(args, "-complete")
	}
	args = append(args, files...)
	if err := mkdir(filepath.Dir(outfile)); err != nil {
		return fmt.Errorf("gc:gc: %v", err)
	}
	return pkg.run(srcdir, nil, t.gc, args...)
}

func (t *gcToolchain) Asm(pkg *Package, srcdir, ofile, sfile string) error {
	includedir := filepath.Join(runtime.GOROOT(), "pkg", "include")
	args := []string{"-o", ofile, "-D", "GOOS_" + t.goos, "-D", "GOARCH_" + t.goarch, "-I", includedir, sfile}
	if err := mkdir(filepath.Dir(ofile)); err != nil {
		return fmt.Errorf("gc:asm: %v", err)
	}
	return pkg.run(srcdir, nil, t.as, args...)
}

func (t *gcToolchain) Ld(pkg *Package, searchpaths, ldflags []string, outfile, afile string) error {
	args := append(ldflags, "-o", outfile)
	for _, d := range searchpaths {
		args = append(args, "-L", d)
	}
	args = append(args, "-extld="+gcc(), "-buildmode=exe")
	args = append(args, afile)
	if err := mkdir(filepath.Dir(outfile)); err != nil {
		return fmt.Errorf("gc:ld: %v", err)
	}
	return pkg.run(".", nil, t.ld, args...)
}

func (t *gcToolchain) Cc(pkg *Package, ofile, cfile string) error {
	return fmt.Errorf("gc15 does not support cc")
}
