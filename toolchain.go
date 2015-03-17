package gb

import (
	"os/exec"
	"strings"
)

// Toolchain represents a standardised set of command line tools
// used to build and test Go programs.
type Toolchain interface {
	Gc(importpath, srcdir, outfile string, files []string) error
	Asm(srcdir, ofile, sfile string) error
	Pack(string, ...string) error
	Ld(string, string) error
	Cc(srcdir, objdir, ofile, cfile string) error

	//	Cgo(string, []string) error
	//	Gcc(string, []string) error
	//	Libgcc() (string, error)
}

func run(dir, command string, args ...string) error {
	_, err := runOut(dir, command, args...)
	return err
}

func runOut(dir, command string, args ...string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	Debugf("cd %s; %s %s", dir, command, strings.Join(args, " "))
	if err != nil {
		Errorf("%s", output)
	}
	return output, err
}

type NullToolchain struct{}

func (NullToolchain) Gc(importpath, srcdir, outfile string, files []string) error {
	Debugf("null:gc %v %v %v %v", importpath, srcdir, outfile, files)
	return nil
}

func (NullToolchain) Asm(srcdir, ofile, sfile string) error {
	Debugf("null:asm %v %v %v", srcdir, ofile, sfile)
	return nil
}
func (NullToolchain) Pack(afile string, ofiles ...string) error {
	Debugf("null:pack %v %v", afile, ofiles)
	return nil
}
func (NullToolchain) Ld(aout string, afile string) error {
	Debugf("null:ld %v %v", aout, afile)
	return nil
}
func (NullToolchain) Cc(srcdir, objdir, ofile, cfile string) error {
	Debugf("null:cc %v %v %v %v", srcdir, objdir, ofile, cfile)
	return nil
}
