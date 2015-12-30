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
	registerCommand(GenerateCmd)
}

var GenerateCmd = &cmd.Command{
	Name:      "generate",
	UsageLine: "generate [-run regexp] [file.go... | packages]",
	Short:     "generate Go files by processing source",
	Long: `
Generate runs commands described by directives within existing files.

Those commands can run any process, but the intent is to create or update Go
source files, for instance by running yacc.

See 'go help generate'.
`,
	Run: func(ctx *gb.Context, args []string) error {
		env := cmd.MergeEnv(os.Environ(), map[string]string{
			"GOPATH": fmt.Sprintf("%s:%s", ctx.Projectdir(), filepath.Join(ctx.Projectdir(), "vendor")),
		})

		goBinary, err := lookupGo()
		if err != nil {
			return err
		}

		args = append([]string{goBinary, "generate"}, args...)

		cmd := exec.Cmd{
			Path: goBinary,
			Args: args,
			Env:  env,

			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}

		return cmd.Run()
	},
}
