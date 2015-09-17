package gb

import (
	"fmt"
	"go/build"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// gc toolchain

type gcToolchain struct {
	gc, cc, ld, as, pack string

	run    func(string, []string, string, ...string) error
	runOut func(io.Writer, string, []string, string, ...string) error
}

func GcToolchain() func(c *Context) error {
	return func(c *Context) error {
		// TODO(dfc) this should come from the context, not the runtime.
		goroot := runtime.GOROOT()

		if gc14 && (c.gohostos != c.gotargetos || c.gohostarch != c.gotargetarch) {
			// cross-compliation is not supported yet #31
			return fmt.Errorf("cross compilation from host %s/%s to target %s/%s not supported with Go 1.4", c.gohostos, c.gohostarch, c.gotargetos, c.gotargetarch)
		}

		tooldir := filepath.Join(goroot, "pkg", "tool", c.gohostos+"_"+c.gohostarch)
		exe := ""
		if c.gohostos == "windows" {
			exe += ".exe"
		}
		switch {
		case gc14:
			archchar, err := build.ArchChar(c.gotargetarch)
			if err != nil {
				return err
			}
			c.tc = &gcToolchain{
				gc:     filepath.Join(tooldir, archchar+"g"+exe),
				ld:     filepath.Join(tooldir, archchar+"l"+exe),
				as:     filepath.Join(tooldir, archchar+"a"+exe),
				cc:     filepath.Join(tooldir, archchar+"c"+exe),
				pack:   filepath.Join(tooldir, "pack"+exe),
				run:    run,
				runOut: runOut,
			}
		case gc15:
			c.tc = &gcToolchain{
				gc:     filepath.Join(tooldir, "compile"+exe),
				ld:     filepath.Join(tooldir, "link"+exe),
				as:     filepath.Join(tooldir, "asm"+exe),
				pack:   filepath.Join(tooldir, "pack"+exe),
				run:    run,
				runOut: runOut,
			}
		default:
			return fmt.Errorf("unsupported Go version: %v", runtime.Version)
		}
		return nil
	}
}

func (t *gcToolchain) Asm(pkg *Package, srcdir, ofile, sfile string) error {
	args := []string{"-o", ofile, "-D", "GOOS_" + pkg.gotargetos, "-D", "GOARCH_" + pkg.gotargetarch}
	switch {
	case gc14:
		includedir := filepath.Join(pkg.Context.Context.GOROOT, "pkg", pkg.gotargetos+"_"+pkg.gotargetarch)
		args = append(args, "-I", includedir)
	case gc15:
		odir := filepath.Join(filepath.Dir(ofile))
		includedir := filepath.Join(runtime.GOROOT(), "pkg", "include")
		args = append(args, "-I", odir, "-I", includedir)
	default:
		return fmt.Errorf("unsupported Go version: %v", runtime.Version)
	}
	args = append(args, sfile)
	if err := mkdir(filepath.Dir(ofile)); err != nil {
		return fmt.Errorf("gc:asm: %v", err)
	}
	return t.run(srcdir, nil, t.as, args...)
}

func (t *gcToolchain) Ld(pkg *Package, searchpaths []string, outfile, afile string) error {
	args := append(pkg.ldflags, "-o", outfile)
	for _, d := range searchpaths {
		args = append(args, "-L", d)
	}
	if gc15 {
		args = append(args, "-buildmode", pkg.buildmode)
	}
	args = append(args, afile)
	if err := mkdir(filepath.Dir(outfile)); err != nil {
		return fmt.Errorf("gc:ld: %v", err)
	}
	return t.run(".", nil, t.ld, args...)
}

func (t *gcToolchain) Cc(pkg *Package, ofile, cfile string) error {
	if gc15 {
		return fmt.Errorf("gc15 does not support cc")
	}
	args := []string{
		"-F", "-V", "-w",
		"-trimpath", pkg.Workdir(),
		"-I", Workdir(pkg),
		"-I", filepath.Join(pkg.Context.Context.GOROOT, "pkg", pkg.gohostos+"_"+pkg.gohostarch), // for runtime.h
		"-o", ofile,
		"-D", "GOOS_" + pkg.gotargetos,
		"-D", "GOARCH_" + pkg.gotargetarch,
		cfile,
	}
	return t.run(pkg.Dir, nil, t.cc, args...)
}

func (t *gcToolchain) Pack(pkg *Package, afiles ...string) error {
	args := []string{"r"}
	args = append(args, afiles...)
	dir := filepath.Dir(afiles[0])
	return t.run(dir, nil, t.pack, args...)
}

func (t *gcToolchain) compiler() string { return t.gc }
func (t *gcToolchain) linker() string   { return t.ld }

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
	} else if gc15 {
		asmhdr := filepath.Join(filepath.Dir(outfile), pkg.Name, "go_asm.h")
		args = append(args, "-asmhdr", asmhdr)
	}

	relativeFiles, err := relativizePaths(srcdir, files)
	if err != nil {
		return err
	}

	args = append(args, relativeFiles...)
	if err := mkdir(filepath.Join(filepath.Dir(outfile), pkg.Name)); err != nil {
		return fmt.Errorf("gc:gc: %v", err)
	}
	return t.runOut(os.Stdout, ".", nil, t.gc, args...)
}

// relativizePaths takes a base path and set of paths relative to that base path
// and returns an equivalent slice of paths that are relative to the current
// working directory
//
// e.g.
// basePath = /x/y
// paths = q.go z.go
// cwd = /x
// returns: [y/q.go, y/z.go]
func relativizePaths(basePath string, paths []string) ([]string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	relativePaths := make([]string, len(paths))
	for i, p := range paths {
		// don't muck with absolute paths
		if filepath.IsAbs(p) {
			relativePaths[i] = p
			continue
		}

		relativePaths[i], err = filepath.Rel(cwd, filepath.Join(basePath, p))
		if err != nil {
			return nil, err
		}
	}
	return relativePaths, nil
}
