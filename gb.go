// Package gb is a tool kit for compiling and testing Go programs.
//
// The executable, cmd/gb, is located in the respective subdirectory
// along with several plugin programs.
package gb

import (
	"bytes"
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
	Ld([]string, []string, string, string) error

	//	Cgo(string, []string) error
	//	Gcc(string, []string) error
	//	Libgcc() (string, error)
}

func mktmpdir() string {
	d, err := ioutil.TempDir("", "gb")
	if err != nil {
		Fatalf("could not create temporary directory: %v", err)
	}
	return d
}

func mkdir(path string) error {
	return os.MkdirAll(path, 0755)
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
	var buf bytes.Buffer
	return runOut(&buf, dir, command, args...)
}

func runOut(output io.Writer, dir, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	cmd.Stdout = output
	cmd.Stderr = os.Stderr
	Debugf("cd %s; %s", cmd.Dir, cmd.Args)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("# %s\n%s", strings.Join(cmd.Args, " "), output)
	}
	return err
}

// joinlist joins a []string representing path items
// using the operating system specific list separator.
func joinlist(l []string) string {
	return strings.Join(l, string(filepath.ListSeparator))
}

func splitQuotedFields(s string) ([]string, error) {
	// Split fields allowing '' or "" around elements.
	// Quotes further inside the string do not count.
	var f []string
	for len(s) > 0 {
		for len(s) > 0 && isWhitespace(s[0]) {
			s = s[1:]
		}
		if len(s) == 0 {
			break
		}
		// Accepted quoted string. No unescaping inside.
		if s[0] == '"' || s[0] == '\'' {
			quote := s[0]
			s = s[1:]
			i := 0
			for i < len(s) && s[i] != quote {
				i++
			}
			if i >= len(s) {
				return nil, fmt.Errorf("unterminated %c string", quote)
			}
			f = append(f, s[:i])
			s = s[i+1:]
			continue
		}
		i := 0
		for i < len(s) && !isWhitespace(s[i]) {
			i++
		}
		f = append(f, s[:i])
		s = s[i:]
	}
	return f, nil
}

func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}
