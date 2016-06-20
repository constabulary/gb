package main

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/constabulary/gb/internal/debug"
)

type Context interface {
	Projectdir() string
	AllPackages(string) ([]string, error)
}

// importPathsNoDotExpansion returns the import paths to use for the given
// command line, but it does no ... expansion.
func importPathsNoDotExpansion(ctx Context, cwd string, args []string) []string {
	srcdir, _ := filepath.Rel(filepath.Join(ctx.Projectdir(), "src"), cwd)
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

		if a == "all" || a == "std" {
			pkgs, err := ctx.AllPackages(a)
			if err != nil {
				fatalf("could not load all packages: %v", err)
			}
			out = append(out, pkgs...)
			continue
		}
		a = path.Join(srcdir, path.Clean(a))
		out = append(out, a)
	}
	return out
}

// importPaths returns the import paths to use for the given command line.
func importPaths(ctx Context, cwd string, args []string) []string {
	args = importPathsNoDotExpansion(ctx, cwd, args)
	var out []string
	for _, a := range args {
		if strings.Contains(a, "...") {
			pkgs, err := ctx.AllPackages(a)
			if err != nil {
				fatalf("could not load all packages: %v", err)
			}
			out = append(out, pkgs...)
			continue
		}
		out = append(out, a)
	}
	return out
}
