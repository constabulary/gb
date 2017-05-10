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
	registerCommand(generateCmd)
}

var generateCmd = &cmd.Command{
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
			"GOPATH": fmt.Sprintf("%s%c%s", ctx.Projectdir(), filepath.ListSeparator, filepath.Join(ctx.Projectdir(), "vendor")),
			"PATH":   fmt.Sprintf("%s%c%s", filepath.Join(ctx.Projectdir(), "bin"), filepath.ListSeparator, os.Getenv("PATH")),
		})

		cmd := exec.Command("go", append([]string{"generate"}, args...)...)
		cmd.Env = env
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		return cmd.Run()
	},
}
