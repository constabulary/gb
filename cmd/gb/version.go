package main

import (
	"fmt"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
)

func init() {
	registerCommand(&cmd.Command{
		Name:      "version",
		UsageLine: `version`,
		Short:     "show current version of gb",
		Run: func(ctx *gb.Context, args []string) (err error) {
			fmt.Println("0.1.1")
			return
		},
		ParseArgs: func(_ *gb.Context, _ string, args []string) []string { return args },
	})
}
