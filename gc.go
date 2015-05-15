package gb

// gc toolchain

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"runtime"
)

type gcToolchain struct {
	goos, goarch, goroot string
	gc, cc, ld, as, pack string
}

type gcoption struct {
	goroot, goos, goarch string
}

// Goroot configures GcToolchain.
func Goroot(goroot string) func(*gcoption) {
	return func(opts *gcoption) {
		opts.goroot = goroot
	}
}

func GcToolchain(opts ...func(*gcoption)) func(c *Context) error {
	defaults := []func(*gcoption){
		func(opt *gcoption) {
			opt.goroot = runtime.GOROOT()
		},
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
		goroot := options.goroot
		goos := options.goos
		goarch := options.goarch
		archchar, err := build.ArchChar(goarch)
		if err != nil {
			return err
		}
		tooldir := filepath.Join(goroot, "pkg", "tool", goos+"_"+goarch)
		c.tc = &gcToolchain{
			goos:   goos,
			goarch: goarch,
			goroot: goroot,
			gc:     filepath.Join(tooldir, archchar+"g"),
			ld:     filepath.Join(tooldir, archchar+"l"),
			as:     filepath.Join(tooldir, archchar+"a"),
			pack:   filepath.Join(tooldir, "pack"),
		}
		return nil
	}
}

func (t *gcToolchain) Gc(searchpaths []string, importpath, srcdir, outfile string, files []string, complete bool) error {
	Debugf("gc:gc %v %v %v %v", importpath, srcdir, outfile, files)

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

func (t *gcToolchain) Pack(afiles ...string) error {
	args := []string{"r"}
	args = append(args, afiles...)
	dir := filepath.Dir(afiles[0])
	return run(dir, t.pack, args...)
}

func (t *gcToolchain) Asm(srcdir, ofile, sfile string) error {
	// TODO(dfc) this is the go 1.4 include path, go 1.5 moves the path to $GOROOT/pkg/include
	includedir := filepath.Join(t.goroot, "pkg", t.goos+"_"+t.goarch)
	args := []string{"-o", ofile, "-D", "GOOS_" + t.goos, "-D", "GOARCH_" + t.goarch, "-I", includedir, sfile}
	err := os.MkdirAll(filepath.Dir(ofile), 0755)
	if err != nil {
		return fmt.Errorf("gc:asm: %v", err)
	}
	return run(srcdir, t.as, args...)
}

func (t *gcToolchain) Ld(searchpaths, ldflags []string, outfile, afile string) error {
	args := append(ldflags, "-o", outfile)
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
