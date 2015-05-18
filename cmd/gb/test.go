package main

import (
	"time"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
)

func init() {
	registerCommand("test", TestCmd)
}

var TestCmd = &cmd.Command{
	ShortDesc: "test a package",
	Run: func(ctx *gb.Context, args []string) error {
		t0 := time.Now()
		ctx.Force = F
		ctx.SkipInstall = FF
		defer func() {
			gb.Debugf("test duration: %v %v", time.Since(t0), ctx.Statistics.String())
		}()

		pkgs, err := cmd.ResolvePackagesWithTests(ctx, args...)
		if err != nil {
			return err
		}
		if err := cmd.Test(pkgs...); err != nil {
			return err
		}
		return ctx.Destroy()
	},
	AddFlags: addBuildFlags,
}
