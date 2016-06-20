package match

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/constabulary/gb/internal/debug"
)

type Context interface {
	AllPackages(string) ([]string, error)
}

// importPathsNoDotExpansion returns the import paths to use for the given
// command line, but it does no ... expansion.
func importPathsNoDotExpansion(srcdir string, cwd string, args []string) []string {
	srcdir, _ = filepath.Rel(srcdir, cwd)
	debug.Debugf("%s %s", cwd, srcdir)
	if srcdir == ".." {
		srcdir = "."
	}
	if len(args) == 0 {
		args = []string{"..."}
	}
	var out []string
	for _, a := range args {
		// Arguments are supposed to be import paths, but
		// as a courtesy to Windows developers, rewrite \ to /
		// in command-line arguments.  Handles .\... and so on.
		if filepath.Separator == '\\' {
			a = strings.Replace(a, `\`, `/`, -1)
		}
		a = path.Join(srcdir, path.Clean(a))
		out = append(out, a)
	}
	return out
}

// ImportPaths returns the import paths to use for the given command line.
func ImportPaths(srcdir string, ctx Context, cwd string, args []string) []string {
	args = importPathsNoDotExpansion(srcdir, cwd, args)
	var out []string
	for _, a := range args {
		if strings.Contains(a, "...") {
			pkgs, err := ctx.AllPackages(a)
			if err != nil {
				fmt.Println("could not load all packages: %v", err)
			}
			out = append(out, pkgs...)
			continue
		}
		out = append(out, a)
	}
	return out
}
