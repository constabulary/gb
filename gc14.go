// +build !go1.5

package gb

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func GcToolchain(opts ...func(*gcoption)) func(c *Context) error {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	defaults := []func(*gcoption){
		func(opt *gcoption) {
			opt.goos = runtime.GOOS
			if v := os.Getenv("GOOS"); v != "" {
				opt.goos = v
			}
		},
		func(opt *gcoption) {
			opt.goarch = runtime.GOARCH
			if v := os.Getenv("GOARCH"); v != "" {
				opt.goarch = v
			}
		},
	}
	var options gcoption
	for _, opt := range append(defaults, opts...) {
		opt(&options)
	}

	return func(c *Context) error {
		goroot := runtime.GOROOT()
		archchar, err := build.ArchChar(goarch)
		if err != nil {
			return err
		}
		tooldir := filepath.Join(goroot, "pkg", "tool", goos+"_"+goarch)
		c.tc = &gcToolchain{
			goos:   options.goos,
			goarch: options.goarch,
			gc:     filepath.Join(tooldir, archchar+"g"),
			ld:     filepath.Join(tooldir, archchar+"l"),
			as:     filepath.Join(tooldir, archchar+"a"),
			cc:     filepath.Join(tooldir, archchar+"c"),
			pack:   filepath.Join(tooldir, "pack"),
		}
		return nil
	}
}

func (t *gcToolchain) Gc(pkg *Package, searchpaths []string, importpath, srcdir, outfile string, files []string, complete bool) error {
	Debugf("gc:gc %v %v %v %v", importpath, srcdir, outfile, files)

	args := append(pkg.gcflags, "-p", importpath, "-pack")
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
	return pkg.runOut(os.Stdout, srcdir, nil, t.gc, args...)
}

func (t *gcToolchain) Asm(pkg *Package, srcdir, ofile, sfile string) error {
	includedir := filepath.Join(runtime.GOROOT(), "pkg", t.goos+"_"+t.goarch)
	args := []string{"-o", ofile, "-D", "GOOS_" + t.goos, "-D", "GOARCH_" + t.goarch, "-I", includedir, sfile}
	if err := mkdir(filepath.Dir(ofile)); err != nil {
		return fmt.Errorf("gc:asm: %v", err)
	}
	return pkg.run(srcdir, nil, t.as, args...)
}

func (t *gcToolchain) Ld(pkg *Package, searchpaths []string, outfile, afile string) error {
	if t.goos != runtime.GOOS || t.goarch != runtime.GOARCH {
		i := strings.Index(outfile, ".")
		if i > 0 {
			outfile = fmt.Sprintf("%s-%s-%s%s", outfile[:i], t.goos, t.goarch, outfile[i:])
		} else {
			outfile = fmt.Sprintf("%s-%s-%s", outfile, t.goos, t.goarch)
		}
	}

	args := append(pkg.ldflags, "-o", outfile)
	for _, d := range searchpaths {
		args = append(args, "-L", d)
	}
	args = append(args, "-extld="+gcc()) // TODO(dfc) go 1.5+, "-buildmode=exe")
	args = append(args, afile)
	if err := mkdir(filepath.Dir(outfile)); err != nil {
		return fmt.Errorf("gc:ld: %v", err)
	}
	return pkg.run(".", nil, t.ld, args...)
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
	return pkg.run(pkg.Dir, nil, t.cc, args...)
}
