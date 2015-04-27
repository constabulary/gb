package main

import (
	"time"

	"github.com/constabulary/gb"
)

func init() {
	registerCommand("test", TestCmd)
}

var TestCmd = &Command{
	Run: func(proj *gb.Project, args []string) error {
		t0 := time.Now()
		tc, err := gb.NewGcToolchain(*goroot, *goos, *goarch)
		if err != nil {
			gb.Fatalf("unable to construct toolchain: %v", err)
		}
		ctx := proj.NewContext(tc)
		ctx.Force = F
		ctx.SkipInstall = FF
		defer func() {
			gb.Infof("test duration: %v %v", time.Since(t0), ctx.Statistics.String())
		}()

		pkgs, err := resolvePackages(ctx, args...)
		if err != nil {
			return err
		}
		if err := gb.Test(pkgs...); err != nil {
			return err
		}
		return ctx.Destroy()
	},
	AddFlags: addBuildFlags,
}
