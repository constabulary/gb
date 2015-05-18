package gb

import (
	"bytes"
	"path/filepath"
	"strings"
)

// cgo support functions

// rungcc1 invokes gcc to compile cfile into ofile
func rungcc1(dir, ofile, cfile string) error {
	gcc := "gcc" // TODO(dfc) handle $CC and clang
	args := []string{
		"-fPIC", "-m64", "-pthread", "-fmessage-length=0",
		"-I", dir,
		"-I", filepath.Dir(ofile),
		"-g", "-O2",
		"-o", ofile,
		"-c", cfile,
	}
	return run(dir, gcc, args...)
}

// rungcc2 links the o files from rungcc1 into a single _cgo_.o.
func rungcc2(dir string, ofiles []string) (string, error) {
	gcc := "gcc" // TODO(dfc) handle $CC and clang
	ofile := filepath.Join(filepath.Dir(ofiles[0]), "_cgo_.o")
	args := []string{
		"-fPIC", "-m64", "-pthread", "-fmessage-length=0",
		"-o", ofile,
	}
	args = append(args, ofiles...)
	args = append(args, "-g", "-O2") // this has to go at the end, because reasons!
	return ofile, run(dir, gcc, args...)
}

// rungcc3 links all previous ofiles together with libgcc into a single _all.o.
func rungcc3(dir string, ofiles []string) (string, error) {
	libgcc, err := libgcc()
	if err != nil {
		return "", nil
	}
	gcc := "gcc" // TODO(dfc) handle $CC and clang
	ofile := filepath.Join(filepath.Dir(ofiles[0]), "_all.o")
	args := []string{
		"-fPIC", "-m64", "-pthread", "-fmessage-length=0",
		"-g", "-O2",
		"-o", ofile,
	}
	args = append(args, ofiles...)
	args = append(args, "-Wl,-r", "-nostdlib", libgcc, "-Wl,--build-id=none")
	return ofile, run(dir, gcc, args...)
}

// libgcc returns the value of gcc -print-libgcc-file-name.
func libgcc() (string, error) {
	gcc := "gcc" // TODO(dfc) handle $CC and clang
	args := []string{
		"-fPIC", "-m64", "-pthread", "-fmessage-length=0",
		"-print-libgcc-file-name",
	}
	var buf bytes.Buffer
	err := runOut(&buf, ".", gcc, args...)
	return strings.TrimSpace(buf.String()), err
}

func cgotool(ctx *Context) string {
	return filepath.Join(ctx.GOROOT, "pkg", "tool", ctx.GOOS+"_"+ctx.GOARCH, "cgo")
}
