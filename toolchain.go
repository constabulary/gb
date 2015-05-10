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

	//	Cgo(string, []string) error
	//	Gcc(string, []string) error
	//	Libgcc() (string, error)
}

// Run returns a Target representing the result of executing a CmdTarget.
func Run(ch chan bool, cmd *exec.Cmd, dep Target) Target {
	annotate := func() error {
		<-ch
		Infof("run %v", cmd.Args)
		err := cmd.Run()
		ch <- true
		if err != nil {
			err = fmt.Errorf("run %v: %v", cmd.Args, err)
		}
		return err
	}
	target := newTarget(annotate, dep)
	return &target // TODO
}
