package cmd

import (
	"path"
	"path/filepath"
	"strings"
)

type Context interface {
	Srcdirs() []string
	AllPackages(string) []string
}

// importPathsNoDotExpansion returns the import paths to use for the given
// command line, but it does no ... expansion.
func importPathsNoDotExpansion(ctx Context, cwd string, args []string) []string {
	srcdir, _ := filepath.Rel(ctx.Srcdirs()[0], cwd)
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
			out = append(out, ctx.AllPackages(a)...)
			continue
		}
		a = path.Join(srcdir, path.Clean(a))
		out = append(out, a)
	}
	return out
}

// importPaths returns the import paths to use for the given command line.
func ImportPaths(ctx Context, cwd string, args []string) []string {
	args = importPathsNoDotExpansion(ctx, cwd, args)
	var out []string
	for _, a := range args {
		if strings.Contains(a, "...") {
			out = append(out, ctx.AllPackages(a)...)
			continue
		}
		out = append(out, a)
	}
	return out
}
