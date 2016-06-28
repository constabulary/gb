package main

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
)

func init() {
	registerCommand(&cmd.Command{
		Name:      "info",
		UsageLine: `info [var ...]`,
		Short:     "info returns information about this project",
		Long: `
info prints gb environment information.

Values:

	GB_PROJECT_DIR
		The root of the gb project.
	GB_SRC_PATH
		The list of gb project source directories.
	GB_PKG_DIR
		The path of the gb project's package cache.
	GB_BIN_SUFFIX
		The suffix applied any binary written to $GB_PROJECT_DIR/bin
	GB_GOROOT
		The value of runtime.GOROOT for the Go version that built this copy of gb.

info returns 0 if the project is well formed, and non zero otherwise.
If one or more variable names is given as arguments, info prints the 
value of each named variable on its own line.
`,
		Run:           info,
		SkipParseArgs: true,
		AddFlags:      addBuildFlags,
	})
}

func info(ctx *gb.Context, args []string) error {
	env := makeenv(ctx)
	// print values for env variables when args are provided
	if len(args) > 0 {
		for _, arg := range args {
			// print each var on its own line, blank line for each invalid variables
			fmt.Println(findenv(env, arg))
		}
		return nil
	}
	// print all variable when no args are provided
	for _, v := range env {
		fmt.Printf("%s=\"%s\"\n", v.name, v.val)
	}
	return nil
}

// joinlist joins path elements using the os specific separator.
// TODO(dfc) it probably gets this wrong on windows in some circumstances.
func joinlist(paths ...string) string {
	return strings.Join(paths, string(filepath.ListSeparator))
}

type envvar struct {
	name, val string
}

func findenv(env []envvar, name string) string {
	for _, e := range env {
		if e.name == name {
			return e.val
		}
	}
	return ""
}

func makeenv(ctx *gb.Context) []envvar {
	return []envvar{
		{"GB_PROJECT_DIR", ctx.Projectdir()},
		{"GB_SRC_PATH", joinlist(
			filepath.Join(ctx.Projectdir(), "src"),
			filepath.Join(ctx.Projectdir(), "vendor", "src"),
		)},
		{"GB_PKG_DIR", ctx.Pkgdir()},
		{"GB_BIN_SUFFIX", ctx.Suffix()},
		{"GB_GOROOT", runtime.GOROOT()},
	}
}
