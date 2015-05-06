package main

import (
	"fmt"
	"go/build"
	"time"

	"github.com/constabulary/gb"
)

func init() {
	registerCommand("test", TestCmd)
}

var TestCmd = &Command{
	ShortDesc: "test a package",
	Run: func(ctx *gb.Context, args []string) error {
		t0 := time.Now()
		ctx.Force = F
		ctx.SkipInstall = FF
		defer func() {
			gb.Infof("test duration: %v %v", time.Since(t0), ctx.Statistics.String())
		}()

		pkgs, err := resolvePackagesWithTests(ctx, args...)
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

// resolvePackagesWithTests is similar to resolvePackages however
// it also loads the test and external test packages of args into
// the context.
func resolvePackagesWithTests(ctx *gb.Context, args ...string) ([]*gb.Package, error) {
	var pkgs []*gb.Package
	for _, arg := range args {
		pkg, err := ctx.ResolvePackageWithTests(arg)
		if err != nil {
			if _, ok := err.(*build.NoGoError); ok {
				gb.Debugf("skipping %q", arg)
				continue
			}
			return pkgs, fmt.Errorf("failed to resolve package %q: %v", arg, err)
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs, nil
}
