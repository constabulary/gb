package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
)

func main() {
	cmd.RunCommand(flag.NewFlagSet("gb-env", flag.ExitOnError), &cmd.Command{
		ShortDesc: "env prints the project environment variables",
		Run:       env,
	}, "", "", nil)
}

func env(ctx *gb.Context, args []string) error {
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "GB_") {
			fmt.Println(e)
		}
	}
	return nil
}
