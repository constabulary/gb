package main

import (
	"flag"
	"time"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
)

func init() {
	registerCommand("test", TestCmd)
}

var (
	testCover bool
	tfs       []string // Arguments passed to the test binary
)

func addTestFlags(fs *flag.FlagSet) {
	addBuildFlags(fs)
	fs.BoolVar(&testCover, "cover", false, "enable coverage analysis")
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
		if err := cmd.Test(cmd.TestFlags(fs, tfs), pkgs...); err != nil {
			return err
		}
		return ctx.Destroy()
	},
	AddFlags: addTestFlags,
	FlagParse: func(flags *flag.FlagSet, args []string) error {
		args, tfs = cmd.TestExtraFlags(fs, args[2:])
		return flags.Parse(args)
	},
}
