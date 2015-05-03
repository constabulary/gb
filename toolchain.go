package gb

import (
	"fmt"
	"os/exec"
)

// Toolchain represents a standardised set of command line tools
// used to build and test Go programs.
type Toolchain interface {
	Gc(searchpaths []string, importpath, srcdir, outfile string, files []string, complete bool) error
	Asm(srcdir, ofile, sfile string) error
	Pack(...string) error
	Ld([]string, string, string) error
	Cc(srcdir, objdir, ofile, cfile string) error

	//	Cgo(string, []string) error
	//	Gcc(string, []string) error
	//	Libgcc() (string, error)
}

// Run returns a Target representing the result of executing a CmdTarget.
func Run(cmd *exec.Cmd, dep Target) Target {
	annotate := func() error {
		Infof("run %v", cmd.Args)
		err := cmd.Run()
		if err != nil {
			err = fmt.Errorf("run %v: %v", cmd.Args, err)
		}
		return err
	}
	target := newTarget(annotate, dep)
	return &target // TODO
}

type nulltoolchain struct{}

func (nulltoolchain) Gc(importpath []string, srcdir, _, outfile string, files []string, _ bool) error {
	Debugf("null:gc %v %v %v %v", importpath, srcdir, outfile, files)
	return nil
}

func (nulltoolchain) Asm(srcdir, ofile, sfile string) error {
	Debugf("null:asm %v %v %v", srcdir, ofile, sfile)
	return nil
}
func (nulltoolchain) Pack(afiles ...string) error {
	Debugf("null:pack %v %v", afiles)
	return nil
}
func (nulltoolchain) Ld(_ []string, aout string, afile string) error {
	Debugf("null:ld %v %v", aout, afile)
	return nil
}
func (nulltoolchain) Cc(srcdir, objdir, ofile, cfile string) error {
	Debugf("null:cc %v %v %v %v", srcdir, objdir, ofile, cfile)
	return nil
}
