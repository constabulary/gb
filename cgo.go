package gb

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// cgo support functions

// rungcc1 invokes gcc to compile cfile into ofile
func rungcc1(pkg *Package, cgoCFLAGS []string, ofile, cfile string) error {
	args := []string{"-g", "-O2", "-fPIC", "-m64", "-pthread", "-fmessage-length=0",
		"-I", pkg.Dir,
		"-I", filepath.Dir(ofile),
	}
	args = append(args, cgoCFLAGS...)
	args = append(args,
		"-o", ofile,
		"-c", cfile,
	)
	t0 := time.Now()
	err := run(pkg.Dir, nil, gcc(), args...)
	pkg.Record("gcc1", time.Since(t0))
	return err
}

// rungcc2 links the o files from rungcc1 into a single _cgo_.o.
func rungcc2(pkg *Package, cgoCFLAGS, cgoLDFLAGS []string, ofile string, ofiles []string) error {
	args := []string{
		"-fPIC", "-m64", "-fmessage-length=0",
	}
	if !isClang() {
		args = append(args, "-pthread")
	}
	args = append(args, "-o", ofile)
	args = append(args, ofiles...)
	args = append(args, cgoLDFLAGS...) // this has to go at the end, because reasons!
	t0 := time.Now()
	err := run(pkg.Dir, nil, gcc(), args...)
	pkg.Record("gcc2", time.Since(t0))
	return err
}

// rungcc3 links all previous ofiles together with libgcc into a single _all.o.
func rungcc3(ctx *Context, dir string, ofile string, ofiles []string) error {
	args := []string{
		"-fPIC", "-m64", "-fmessage-length=0",
	}
	if !isClang() {
		args = append(args, "-pthread")
	}
	args = append(args, "-g", "-O2", "-o", ofile)
	args = append(args, ofiles...)
	args = append(args, "-Wl,-r", "-nostdlib")
	if !isClang() {
		libgcc, err := libgcc(ctx)
		if err != nil {
			return nil
		}
		args = append(args, libgcc, "-Wl,--build-id=none")
	}
	t0 := time.Now()
	err := run(dir, nil, gcc(), args...)
	ctx.Record("gcc3", time.Since(t0))
	return err
}

// libgcc returns the value of gcc -print-libgcc-file-name.
func libgcc(ctx *Context) (string, error) {
	args := []string{
		"-print-libgcc-file-name",
	}
	var buf bytes.Buffer
	err := runOut(&buf, ".", nil, gcc(), args...)
	return strings.TrimSpace(buf.String()), err
}

func cgotool(ctx *Context) string {
	return filepath.Join(ctx.GOROOT, "pkg", "tool", ctx.GOOS+"_"+ctx.GOARCH, "cgo")
}

func gcc() string {
	return gccBaseCmd()[0] // TODO(dfc) handle gcc wrappers properly
}

func isClang() bool {
	return strings.HasPrefix(gcc(), "clang")
}

// gccBaseCmd returns the start of the compiler command line.
// It uses $CC if set, or else $GCC, or else the default
// compiler for the operating system is used.
func gccBaseCmd() []string {
	// Use $CC if set, since that's what the build uses.
	if ret := strings.Fields(os.Getenv("CC")); len(ret) > 0 {
		return ret
	}
	// Try $GCC if set, since that's what we used to use.
	if ret := strings.Fields(os.Getenv("GCC")); len(ret) > 0 {
		return ret
	}
	return strings.Fields(defaultCC)
}

// envList returns the value of the given environment variable broken
// into fields, using the default value when the variable is empty.
func envList(key, def string) []string {
	v := os.Getenv(key)
	if v == "" {
		v = def
	}
	return strings.Fields(v)
}

// Return the flags to use when invoking the C or C++ compilers, or cgo.
func cflags(p *Package, def bool) (cppflags, cflags, cxxflags, ldflags []string) {
	var defaults string
	if def {
		defaults = "-g -O2"
	}

	cppflags = stringList(envList("CGO_CPPFLAGS", ""), p.CgoCPPFLAGS)
	cflags = stringList(envList("CGO_CFLAGS", defaults), p.CgoCFLAGS)
	cxxflags = stringList(envList("CGO_CXXFLAGS", defaults), p.CgoCXXFLAGS)
	ldflags = stringList(envList("CGO_LDFLAGS", defaults), p.CgoLDFLAGS)
	return
}

// call pkg-config and return the cflags and ldflags.
func pkgconfig(p *Package) ([]string, []string, error) {
	if len(p.CgoPkgConfig) == 0 {
		return nil, nil, nil // nothing to do
	}
	args := []string{
		"--cflags",
	}
	args = append(args, p.CgoPkgConfig...)
	var out bytes.Buffer
	err := runOut(&out, p.Dir, nil, "pkg-config", args...)
	if err != nil {
		return nil, nil, err
	}
	cflags := strings.Fields(out.String())

	args = []string{
		"--libs",
	}
	args = append(args, p.CgoPkgConfig...)
	out.Reset()
	err = runOut(&out, p.Dir, nil, "pkg-config", args...)
	if err != nil {
		return nil, nil, err
	}
	ldflags := strings.Fields(out.String())
	return cflags, ldflags, nil
}

func quoteFlags(flags []string) []string {
	quoted := make([]string, len(flags))
	for i, f := range flags {
		quoted[i] = strconv.Quote(f)
	}
	return quoted
}
