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
)

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
		err = fmt.Errorf("%v: %s\n%s", cmd.Args, err, output)
	}
	return output, err
}
