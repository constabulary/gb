package gb

import (
	"bytes"
	"fmt"
	"go/build"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

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

		if goversion == 1.4 && (c.gohostos != c.gotargetos || c.gohostarch != c.gotargetarch) {
			// cross-compliation is not supported yet #31
			return errors.Errorf("cross compilation from host %s/%s to target %s/%s not supported with Go 1.4", c.gohostos, c.gohostarch, c.gotargetos, c.gotargetarch)
		}

		tooldir := filepath.Join(goroot, "pkg", "tool", c.gohostos+"_"+c.gohostarch)
		exe := ""
		if c.gohostos == "windows" {
			exe += ".exe"
		}
		switch {
		case goversion == 1.4:
			archchar, err := build.ArchChar(c.gotargetarch)
			if err != nil {
				return err
			}
			c.tc = &gcToolchain{
				gc:   filepath.Join(tooldir, archchar+"g"+exe),
				ld:   filepath.Join(tooldir, archchar+"l"+exe),
				as:   filepath.Join(tooldir, archchar+"a"+exe),
				cc:   filepath.Join(tooldir, archchar+"c"+exe),
				pack: filepath.Join(tooldir, "pack"+exe),
			}
		case goversion > 1.4:
			c.tc = &gcToolchain{
				gc:   filepath.Join(tooldir, "compile"+exe),
				ld:   filepath.Join(tooldir, "link"+exe),
				as:   filepath.Join(tooldir, "asm"+exe),
				pack: filepath.Join(tooldir, "pack"+exe),
			}
		default:
			return errors.Errorf("unsupported Go version: %v", runtime.Version)
		}
		return nil
	}
}

func (t *gcToolchain) Asm(pkg *Package, srcdir, ofile, sfile string) error {
	args := []string{"-o", ofile, "-D", "GOOS_" + pkg.gotargetos, "-D", "GOARCH_" + pkg.gotargetarch}
	switch {
	case goversion == 1.4:
		includedir := filepath.Join(runtime.GOROOT(), "pkg", pkg.gotargetos+"_"+pkg.gotargetarch)
		args = append(args, "-I", includedir)
	case goversion > 1.4:
		odir := filepath.Join(filepath.Dir(ofile))
		includedir := filepath.Join(runtime.GOROOT(), "pkg", "include")
		args = append(args, "-I", odir, "-I", includedir)
	default:
		return errors.Errorf("unsupported Go version: %v", runtime.Version)
	}
	args = append(args, sfile)
	if err := mkdir(filepath.Dir(ofile)); err != nil {
		return errors.Errorf("gc:asm: %v", err)
	}
	var buf bytes.Buffer
	err := runOut(&buf, srcdir, nil, t.as, args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "# %s\n", pkg.ImportPath)
		io.Copy(os.Stderr, &buf)
	}
	return err
}

func (t *gcToolchain) Ld(pkg *Package, searchpaths []string, outfile, afile string) error {
	// to ensure we don't write a partial binary, link the binary to a temporary file in
	// in the target directory, then rename.
	dir := filepath.Dir(outfile)
	tmp, err := ioutil.TempFile(dir, ".gb-link")
	if err != nil {
		return err
	}
	tmp.Close()

	args := append(pkg.ldflags, "-o", tmp.Name())
	for _, d := range searchpaths {
		args = append(args, "-L", d)
	}
	args = append(args, "-extld", linkCmd(pkg, "CC", defaultCC))
	if goversion > 1.4 {
		args = append(args, "-buildmode", pkg.buildmode)
	}
	args = append(args, afile)

	if err := mkdir(dir); err != nil {
		return err
	}

	var buf bytes.Buffer
	if err = runOut(&buf, ".", nil, t.ld, args...); err != nil {
		os.Remove(tmp.Name()) // remove partial file
		fmt.Fprintf(os.Stderr, "# %s\n", pkg.ImportPath)
		io.Copy(os.Stderr, &buf)
		return err
	}
	return os.Rename(tmp.Name(), outfile)
}

func (t *gcToolchain) Cc(pkg *Package, ofile, cfile string) error {
	if goversion > 1.4 {
		return errors.Errorf("gc %f does not support cc", goversion)
	}
	args := []string{
		"-F", "-V", "-w",
		"-trimpath", pkg.Workdir(),
		"-I", Workdir(pkg),
		"-I", filepath.Join(runtime.GOROOT(), "pkg", pkg.gohostos+"_"+pkg.gohostarch), // for runtime.h
		"-o", ofile,
		"-D", "GOOS_" + pkg.gotargetos,
		"-D", "GOARCH_" + pkg.gotargetarch,
		cfile,
	}
	var buf bytes.Buffer
	err := runOut(&buf, pkg.Dir, nil, t.cc, args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "# %s\n", pkg.ImportPath)
		io.Copy(os.Stderr, &buf)
	}
	return err
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

	switch {
	case pkg.Complete():
		args = append(args, "-complete")
	case goversion > 1.4:
		asmhdr := filepath.Join(filepath.Dir(outfile), pkg.Name, "go_asm.h")
		args = append(args, "-asmhdr", asmhdr)
	}

	// If there are vendored components, create an -importmap to map the import statement
	// to the vendored import path. The possibilities for abusing this flag are endless.
	if goversion > 1.5 && pkg.Standard {
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
	err := runOut(&buf, srcdir, nil, t.gc, args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "# %s\n", pkg.ImportPath)
		io.Copy(os.Stderr, &buf)
	}
	return err
}
