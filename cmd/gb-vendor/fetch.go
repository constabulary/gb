package main

import (
	"fmt"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
	"github.com/constabulary/gb/cmd/gb-vendor/vendor"
)

func init() {
	registerCommand("fetch", FetchCmd)
}

var FetchCmd = &cmd.Command{
	ShortDesc: "fetch a remote dependency",
	Run: func(ctx *gb.Context, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("fetch: import path missing")
		}

		_, err := vendor.ReadManifest(manifestFile(ctx))
		if err != nil {
			return fmt.Errorf("could not load manifest: %v", err)
		}

		return nil
	},
}
