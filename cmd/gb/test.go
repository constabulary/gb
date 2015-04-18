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
		defer func() {
			gb.Infof("build duration: %v", time.Since(t0))
		}()

		tc, err := gb.NewGcToolchain(*goroot, *goos, *goarch)
		if err != nil {
			gb.Fatalf("unable to construct toolchain: %v", err)
		}
		//ctx := proj.NewContext(new(gb.NullToolchain))
		ctx := proj.NewContext(tc)
		defer func() {
			gb.Debugf("build statistics: %v", ctx.Statistics.String())
		}()
		pkgs, err := resolvePackages(ctx, args...)
		if err != nil {
			return err
		}
		if err := execute(gb.Test, pkgs...); err != nil {
			return err
		}
		return ctx.Destroy()
	},
	AddFlags: addBuildFlags,
}
