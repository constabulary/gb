package gb

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
)

// cgo support functions

type cgoTarget string

func (t cgoTarget) Objfile() string { return string(t) }
func (t cgoTarget) Result() error   { return nil }

// rungcc1 invokes gcc to compile cfile into ofile
func rungcc1(dir, ofile, cfile string) error {
	args := []string{
		"-fPIC", "-m64", "-pthread", "-fmessage-length=0",
		"-I", dir,
		"-I", filepath.Dir(ofile),
		"-g", "-O2",
		"-o", ofile,
		"-c", cfile,
	}
	return run(dir, gcc(), args...)
}

// rungcc2 links the o files from rungcc1 into a single _cgo_.o.
func rungcc2(dir string, ofile string, ofiles []string) error {
	args := []string{
		"-fPIC", "-m64", "-pthread", "-fmessage-length=0",
		"-o", ofile,
	}
	args = append(args, ofiles...)
	args = append(args, "-g", "-O2") // this has to go at the end, because reasons!
	return run(dir, gcc(), args...)
}

// rungcc3 links all previous ofiles together with libgcc into a single _all.o.
func rungcc3(dir string, ofiles []string) (string, error) {
	libgcc, err := libgcc()
	if err != nil {
		return "", nil
	}
	ofile := filepath.Join(filepath.Dir(ofiles[0]), "_all.o")
	args := []string{
		"-fPIC", "-m64", "-pthread", "-fmessage-length=0",
		"-g", "-O2",
		"-o", ofile,
	}
	args = append(args, ofiles...)
	args = append(args, "-Wl,-r", "-nostdlib", libgcc, "-Wl,--build-id=none")
	return ofile, run(dir, gcc(), args...)
}

// libgcc returns the value of gcc -print-libgcc-file-name.
func libgcc() (string, error) {
	args := []string{
		"-fPIC", "-m64", "-pthread", "-fmessage-length=0",
		"-print-libgcc-file-name",
	}
	var buf bytes.Buffer
	err := runOut(&buf, ".", gcc(), args...)
	return strings.TrimSpace(buf.String()), err
}

func cgotool(ctx *Context) string {
	return filepath.Join(ctx.GOROOT, "pkg", "tool", ctx.GOOS+"_"+ctx.GOARCH, "cgo")
}

func gcc() string {
	return gccBaseCmd()[0] // TODO(dfc) handle gcc wrappers properly
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

// gccMachine returns the gcc -m flag to use, either "-m32", "-m64" or "-marm".
func (t *gcToolchain) gccMachine() []string {
	switch t.goarch {
	case "amd64":
		return []string{"-m64"}
	case "386":
		return []string{"-m32"}
	case "arm":
		return []string{"-marm"} // not thumb
	case "s390":
		return []string{"-m31"}
	case "s390x":
		return []string{"-m64"}
	default:
		return nil
	}
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
