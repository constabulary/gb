package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
	"github.com/pkg/exec"
)

func init() {
	registerCommand(&cmd.Command{
		Name:      "doc",
		UsageLine: `doc <pkg> <sym>[.<method>]`,
		Short:     "show documentation for a package or symbol",
		Run: func(ctx *gb.Context, args []string) error {
			env := cmd.MergeEnv(os.Environ(), map[string]string{
				"GOPATH": fmt.Sprintf("%s:%s", ctx.Projectdir(), filepath.Join(ctx.Projectdir(), "vendor")),
			})
			if len(args) == 0 {
				args = append(args, ".")
			}
			args = append([]string{filepath.Join(ctx.GOROOT, "bin", "godoc")}, args...)

			cmd := exec.Command(args[0], args[1:]...)
			cmd.Cmd.Env = env
			return cmd.Run(
				exec.Stdin(os.Stdin),
				exec.Stdout(os.Stdout),
				exec.Stderr(os.Stderr),
			)
		},
		ParseArgs: func(_ *gb.Context, _ string, args []string) []string { return args },
	})
}
