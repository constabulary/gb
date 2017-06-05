package main

import (
	"fmt"
	"runtime"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
)

const version = "0.4.3"

func init() {
	registerCommand(&cmd.Command{
		Name:      "version",
		UsageLine: `version`,
		Short:     "print the gb version",
		Long: `
Version prints the gb version and the Go version, as reported by runtime.Version.
`,
		Run:           printVersion,
		SkipParseArgs: true,
	})
}

func printVersion(ctx *gb.Context, args []string) error {
	fmt.Printf("gb version %s (go version %s)\n", gbversion, runtime.Version())
	return nil
}
