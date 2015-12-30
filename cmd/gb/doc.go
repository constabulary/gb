package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
)

func init() {
	registerCommand(&cmd.Command{
		Name:      "doc",
		UsageLine: `doc <pkg> <sym>[.<method>]`,
		Short:     "show documentation for a package or symbol",
		Long: `
Doc shows documentation for a package or symbol.

See 'go help doc'.
`,
		Run: func(ctx *gb.Context, args []string) error {
			env := cmd.MergeEnv(os.Environ(), map[string]string{
				"GOPATH": fmt.Sprintf("%s:%s", ctx.Projectdir(), filepath.Join(ctx.Projectdir(), "vendor")),
			})
			if len(args) == 0 {
				args = append(args, ".")
			}

			cmd := exec.Command("godoc", args...)
			cmd.Env = env
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		},
		SkipParseArgs: true,
	})
}
