package main

import (
	"fmt"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
)

func init() {
	registerCommand(EnvCmd)
}

var EnvCmd = &cmd.Command{
	Name:      "env",
	UsageLine: `env`,
	Short:     "print project environment variables",
	Long: `Env prints project environment variables.

`,
	Run: env,
}

func env(ctx *gb.Context, args []string) error {
	env := makeenv(ctx)
	for _, e := range env {
		fmt.Printf("%s=%q\n", e.name, e.val)
	}
	return nil
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
	}
}
