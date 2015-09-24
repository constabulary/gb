package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
)

func init() {
	registerCommand(&cmd.Command{
		Name:      "info",
		UsageLine: `info`,
		Short:     "info returns information about this project",
		Long: `info returns information about this project.

info returns 0 if the project is well formed, and non zero otherwise.
`,
		Run:      info,
		AddFlags: addBuildFlags,
	})
}

func info(ctx *gb.Context, args []string) error {
	fmt.Printf("GB_PROJECT_DIR=%q\n", ctx.Projectdir())
	fmt.Printf("GB_SRC_PATH=%q\n", joinlist(ctx.Srcdirs()...))
	fmt.Printf("GB_PKG_DIR=%q\n", ctx.Pkgdir())
	fmt.Printf("GB_BIN_SUFFIX=%q\n", ctx.Suffix())
	return nil
}

// joinlist joins path elements using the os specific separator.
// TODO(dfc) it probably gets this wrong on windows in some circumstances.
func joinlist(paths ...string) string {
	return strings.Join(paths, string(filepath.ListSeparator))
}
