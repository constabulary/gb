package main

import (
	"os"
	"path/filepath"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
	"github.com/pkg/exec"
)

func init() {
	registerCommand(GenerateCmd)
}

var GenerateCmd = &cmd.Command{
	Name:      "generate",
	UsageLine: "generate",
	Short:     "generate Go files by processing source",
	Long: `Generate runs commands described by directives within existing files.
Those commands can run any process but the intent is to create or update Go
source files, for instance by running yacc.

See 'go help generate'`,
	Run: func(ctx *gb.Context, args []string) error {

		args = append([]string{filepath.Join(ctx.GOROOT, "bin", "go"), "generate"}, args...)

		cmd := exec.Command(args[0], args[1:]...)
		return cmd.Run(
			exec.Setenv("GOPATH", ctx.Projectdir()+":"+filepath.Join(ctx.Projectdir(), "vendor")),
			exec.Stdin(os.Stdin),
			exec.Stdout(os.Stdout),
			exec.Stderr(os.Stderr),
		)
	},
}
