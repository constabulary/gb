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
		tooldir := filepath.Join(runtime.GOROOT(), "pkg", "tool", runtime.GOOS+"_"+runtime.GOARCH)
		c.tc = &gcToolchain{
			goos:   options.goos,
			goarch: options.goarch,
			gc:     filepath.Join(tooldir, "compile"),
			ld:     filepath.Join(tooldir, "link"),
			as:     filepath.Join(tooldir, "asm"),
			pack:   filepath.Join(tooldir, "pack"),
		}
		return nil
	}
}

func (t *gcToolchain) Gc(pkg *Package, searchpaths []string, importpath, srcdir, outfile string, files []string, complete bool) error {
	args := append(pkg.gcflags, "-p", importpath, "-pack")
	args = append(args, "-o", outfile)
	for _, d := range searchpaths {
		args = append(args, "-I", d)
	}
	if complete {
		args = append(args, "-complete")
	}

	// runtime requires special support
	if pkg.Name == "runtime" {
		args = append(args, "-+")
		asmhdr := filepath.Join(filepath.Dir(outfile), "go_asm.h")
		args = append(args, "-asmhdr", asmhdr)
	}

	args = append(args, files...)
	if err := mkdir(filepath.Dir(outfile)); err != nil {
		return fmt.Errorf("gc:gc: %v", err)
	}
	return runOut(os.Stdout, srcdir, nil, t.gc, args...)
}

func (t *gcToolchain) Asm(pkg *Package, srcdir, ofile, sfile string) error {
	includedir := filepath.Join(runtime.GOROOT(), "pkg", "include")
	args := []string{"-o", ofile, "-D", "GOOS_" + t.goos, "-D", "GOARCH_" + t.goarch, "-I", filepath.Dir(filepath.Dir(ofile)), "-I", includedir, sfile}
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
