// Package gb is a tool kit for building Go packages and programs.
//
// The executable, cmd/gb, is located in the respective subdirectory
// along with several plugin programs.
package gb

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/constabulary/gb/log"
)

// Toolchain represents a standardised set of command line tools
// used to build and test Go programs.
type Toolchain interface {
	Gc(pkg *Package, searchpaths []string, importpath, srcdir, outfile string, files []string) error
	Asm(pkg *Package, srcdir, ofile, sfile string) error
	Pack(pkg *Package, afiles ...string) error
	Ld(*Package, []string, string, string) error
	Cc(pkg *Package, ofile string, cfile string) error

	// compiler returns the location of the compiler for .go source code
	compiler() string

	// linker returns the location of the linker for this toolchain
	linker() string
}

// Actions and Tasks.
//
// Actions and Tasks allow gb to separate the role of describing the
// order in which work will be done, from the work itself.
// Actions are the former, they describe the graph of dependencies
// between actions, and thus the work to be done. By traversing the action
// graph, we can do the work, executing Tasks in a sane order.
//
// Tasks describe the work to be done, without being concerned with
// the order in which the work is done -- that is up to the code that
// places Tasks into actions. Tasks also know more intimate details about
// filesystems, processes, file lists, etc, that Actions do not.
//
// Action graphs (they are not strictly trees as branchs converge on base actions)
// contain only work to be performed, there are no Actions with empty Tasks
// or Tasks which do no work.
//
// Actions are executed by Executors, but can also be transformed, mutated,
// or even graphed.

// An Action describes a task to be performed and a set
// of Actions that the task depends on.
type Action struct {

	// Name describes the action.
	Name string

	// Deps identifies the Actions that this Action depends.
	Deps []*Action

	// Task identifies the that this action represents.
	Task
}

// Task represents some work to be performed. It contains a single method
// Run, which is expected to be executed at most once.
type Task interface {

	// Run will initiate the work that this task represents and
	// block until the work is complete.
	Run() error
}

// TaskFn is a Task that can execute itself.
type TaskFn func() error

func (fn TaskFn) Run() error { return fn() }

func mktmpdir() string {
	d, err := ioutil.TempDir("", "gb")
	if err != nil {
		log.Fatalf("could not create temporary directory: %v", err)
	}
	return d
}

func mkdir(path string) error {
	return os.MkdirAll(path, 0755)
}

func copyfile(dst, src string) error {
	err := mkdir(filepath.Dir(dst))
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
	defer w.Close()
	log.Debugf("copyfile(dst: %v, src: %v)", dst, src)
	_, err = io.Copy(w, r)
	return err
}

// joinlist joins a []string representing path items
// using the operating system specific list separator.
func joinlist(l []string) string {
	return strings.Join(l, string(filepath.ListSeparator))
}

// stripext strips the extension from a filename.
// The extension is defined by filepath.Ext.
func stripext(path string) string {
	return path[:len(path)-len(filepath.Ext(path))]
}
