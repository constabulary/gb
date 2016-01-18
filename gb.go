// Package gb is a tool kit for building Go packages and programs.
//
// The executable, cmd/gb, is located in the respective subdirectory
// along with several plugin programs.
package gb

import (
	"go/build"
	"os"
	"path/filepath"
)

var releaseTags = build.Default.ReleaseTags

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

	// Run identifies the task that this action represents.
	Run func() error
}

func mkdir(path string) error {
	return os.MkdirAll(path, 0755)
}

// stripext strips the extension from a filename.
// The extension is defined by filepath.Ext.
func stripext(path string) string {
	return path[:len(path)-len(filepath.Ext(path))]
}

func relImportPath(root, path string) (string, error) {
	if isRel(path) {
		return filepath.Rel(root, path)
	}
	return path, nil
}

// isRel returns if an import path is relative or absolute.
func isRel(path string) bool {
	// TODO(dfc) should this be strings.StartsWith(".")
	return path == "."
}
