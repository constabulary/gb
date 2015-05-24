package main

import (
	"fmt"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
)

func init() {
	registerCommand("env", EnvCmd)
}

var EnvCmd = &cmd.Command{
	ShortDesc: "env prints the project environment variables",
	Run:       env,
}

func env(ctx *gb.Context, args []string) error {
	env := makeenv(ctx)
	if len(args) > 0 {
		for _, arg := range args {
			fmt.Println(findenv(env, arg))
		}
		return nil
	}
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
