package gb

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/constabulary/gb/internal/version"
	"github.com/pkg/errors"
)

// gc toolchain

type gcToolchain struct {
	gc, cc, ld, as, pack string
}

func GcToolchain() func(c *Context) error {
	return func(c *Context) error {
		// TODO(dfc) this should come from the context, not the runtime.
		goroot := runtime.GOROOT()

		tooldir := filepath.Join(goroot, "pkg", "tool", c.gohostos+"_"+c.gohostarch)
		exe := ""
		if c.gohostos == "windows" {
			exe += ".exe"
		}
		switch {
		case version.Version > 1.5:
			c.tc = &gcToolchain{
				gc:   filepath.Join(tooldir, "compile"+exe),
				ld:   filepath.Join(tooldir, "link"+exe),
				as:   filepath.Join(tooldir, "asm"+exe),
				pack: filepath.Join(tooldir, "pack"+exe),
			}
			return nil
		default:
			return errors.Errorf("unsupported Go version: %v", runtime.Version())
		}
	}
}

func (t *gcToolchain) Asm(pkg *Package, ofile, sfile string) error {
	args := []string{"-o", ofile, "-D", "GOOS_" + pkg.gotargetos, "-D", "GOARCH_" + pkg.gotargetarch}
	switch {
	case version.Version > 1.5:
		odir := filepath.Join(filepath.Dir(ofile))
		includedir := filepath.Join(runtime.GOROOT(), "pkg", "include")
		args = append(args, "-I", odir, "-I", includedir)
	default:
		return errors.Errorf("unsupported Go version: %v", runtime.Version())
	}
	args = append(args, sfile)
	if err := mkdir(filepath.Dir(ofile)); err != nil {
		return errors.Errorf("gc:asm: %v", err)
	}
	var buf bytes.Buffer
	err := runOut(&buf, pkg.Dir, nil, t.as, args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "# %s\n", pkg.ImportPath)
		io.Copy(os.Stderr, &buf)
	}
	return err
}

func (t *gcToolchain) Ld(pkg *Package) error {
	// to ensure we don't write a partial binary, link the binary to a temporary file in
	// in the target directory, then rename.
	dir := pkg.bindir()
	if err := mkdir(dir); err != nil {
		return err
	}
	tmp, err := ioutil.TempFile(dir, ".gb-link")
	if err != nil {
		return err
	}
	tmp.Close()

	args := append(pkg.ldflags, "-o", tmp.Name())
	for _, d := range pkg.includePaths() {
		args = append(args, "-L", d)
	}
	args = append(args, "-extld", linkCmd(pkg, "CC", defaultCC))
	args = append(args, "-buildmode", pkg.buildmode)
	args = append(args, pkg.objfile())

	var buf bytes.Buffer
	if err = runOut(&buf, ".", nil, t.ld, args...); err != nil {
		os.Remove(tmp.Name()) // remove partial file
		fmt.Fprintf(os.Stderr, "# %s\n", pkg.ImportPath)
		io.Copy(os.Stderr, &buf)
		return err
	}
	return os.Rename(tmp.Name(), pkg.Binfile())
}

func (t *gcToolchain) Cc(pkg *Package, ofile, cfile string) error {
	return errors.Errorf("gc %f does not support cc", version.Version)
}

func (t *gcToolchain) Pack(pkg *Package, afiles ...string) error {
	args := []string{"r"}
	args = append(args, afiles...)
	dir := filepath.Dir(afiles[0])
	var buf bytes.Buffer
	err := runOut(&buf, dir, nil, t.pack, args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "# %s\n", pkg.ImportPath)
		io.Copy(os.Stderr, &buf)
	}
	return err
}

func (t *gcToolchain) compiler() string { return t.gc }
func (t *gcToolchain) linker() string   { return t.ld }

func (t *gcToolchain) Gc(pkg *Package, files []string) error {
	outfile := pkg.objfile()
	args := append(pkg.gcflags, "-p", pkg.ImportPath, "-pack")
	args = append(args, "-o", outfile)
	for _, d := range pkg.includePaths() {
		args = append(args, "-I", d)
	}
	if pkg.Goroot && pkg.ImportPath == "runtime" {
		// runtime compiles with a special gc flag to emit
		// additional reflect type data.
		args = append(args, "-+")
	}

	if pkg.complete() {
		args = append(args, "-complete")
	} else {
		asmhdr := filepath.Join(filepath.Dir(outfile), pkg.Name, "go_asm.h")
		args = append(args, "-asmhdr", asmhdr)
	}

	// If there are vendored components, create an -importmap to map the import statement
	// to the vendored import path. The possibilities for abusing this flag are endless.
	if pkg.Goroot {
		for _, path := range pkg.Package.Imports {
			if i := strings.LastIndex(path, "/vendor/"); i >= 0 {
				args = append(args, "-importmap", path[i+len("/vendor/"):]+"="+path)
			} else if strings.HasPrefix(path, "vendor/") {
				args = append(args, "-importmap", path[len("vendor/"):]+"="+path)
			}
		}
	}

	args = append(args, files...)
	if err := mkdir(filepath.Join(filepath.Dir(outfile), pkg.Name)); err != nil {
		return errors.Wrap(err, "mkdir")
	}
	var buf bytes.Buffer
	err := runOut(&buf, pkg.Dir, nil, t.gc, args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "# %s\n", pkg.ImportPath)
		io.Copy(os.Stderr, &buf)
	}
	return err
}
