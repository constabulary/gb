package main

import (
	"flag"
	"fmt"
	"go/build"
	"time"

	"github.com/constabulary/gb"
)

func init() {
	registerCommand("build", BuildCmd)
}

var (
	// build flags

	// should we build all packages in this project.
	// defaults to true when build is invoked from the project root.
	A bool

	// should we perform a release build +release tag ?
	// defaults to false, +debug.
	R bool

	// force rebuild of packages
	F bool

	// skip caching of packages
	FF bool
)

func addBuildFlags(fs *flag.FlagSet) {
	// TODO(dfc) this should accept a *gb.Context
	fs.BoolVar(&A, "a", false, "build all packages in this project")
	fs.BoolVar(&R, "r", false, "perform a release build")
	fs.BoolVar(&F, "f", false, "rebuild up to date packages")
	fs.BoolVar(&FF, "F", false, "do not cache built packages")
}

var BuildCmd = &Command{
	ShortDesc: "build a package",
	Run: func(ctx *gb.Context, args []string) error {
		// TODO(dfc) run should take a *gb.Context not a *gb.Project
		t0 := time.Now()
		ctx.Force = F
		ctx.SkipInstall = FF
		defer func() {
			gb.Infof("build duration: %v %v", time.Since(t0), ctx.Statistics.String())
		}()

		pkgs, err := resolvePackages(ctx, args...)
		if err != nil {
			return err
		}
		if err := gb.Build(pkgs...); err != nil {
			return err
		}
		return ctx.Destroy()
	},
	AddFlags: addBuildFlags,
}

func resolvePackages(ctx *gb.Context, args ...string) ([]*gb.Package, error) {
	var pkgs []*gb.Package
	for _, arg := range args {
		pkg, err := ctx.ResolvePackage(arg)
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
