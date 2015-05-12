// Package gb is a tool kit for compiling and testing Go programs.
//
// The executable, cmd/gb, is located in the respective subdirectory
// along with several plugin programs.
package gb

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func mktmpdir() string {
	d, err := ioutil.TempDir("", "gb")
	if err != nil {
		Fatalf("could not create temporary directory: %v", err)
	}
	return d
}

func copyfile(dst, src string) error {
	err := os.MkdirAll(filepath.Dir(dst), 0755)
	if err != nil {
		return fmt.Errorf("copyfile: mkdirall: %v", err)
	}
	r, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("copyfile: open(%q): %v", src, err)
	}
	defer r.Close()
	w, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("copyfile: create(%q): %v", dst, err)
	}
	Debugf("copyfile(dst: %v, src: %v)", dst, src)
	_, err = io.Copy(w, r)
	return err
}

func run(dir, command string, args ...string) error {
	_, err := runOut(dir, command, args...)
	return err
}

func runOut(dir, command string, args ...string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	Debugf("cd %s; %s", cmd.Dir, cmd.Args)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("# %s\n%s", strings.Join(cmd.Args, " "), output)
	}
	return output, err
}

// joinlist joins a []string representing path items
// using the operating system specific list separator.
func joinlist(l []string) string {
	return strings.Join(l, string(filepath.ListSeparator))
}
