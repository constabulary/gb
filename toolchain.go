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

	Cgo(string, []string) error
	Gcc(string, []string) error
	Libgcc() (string, error)

	name() string
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
