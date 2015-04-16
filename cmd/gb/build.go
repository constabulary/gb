package main

import (
	"flag"
	"fmt"
	"go/build"
	"path/filepath"
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
)

func addBuildFlags(fs *flag.FlagSet) {
	fs.BoolVar(&A, "a", false, "build all packages in this project")
	fs.BoolVar(&R, "r", false, "perform a release build")
}

var BuildCmd = &Command{
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
		results := make(chan gb.Target, len(pkgs))
		go func() {
			defer close(results)
			for _, pkg := range pkgs {
				results <- gb.Build(pkg)
			}
		}()
		for result := range results {
			if err := result.Result(); err != nil {
				return err
			}
		}
		return ctx.Destroy()
	},
	AddFlags: addBuildFlags,
}

func resolvePackages(ctx *gb.Context, args ...string) ([]*gb.Package, error) {
	var pkgs []*gb.Package
	for _, arg := range args {
		if arg == "." {
			var err error
			arg, err = filepath.Rel(ctx.Srcdirs()[0], mustGetwd())
			if err != nil {
				return pkgs, err
			}
		}
		pkg := ctx.ResolvePackage(arg)
		if err := pkg.Result(); err != nil {
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
