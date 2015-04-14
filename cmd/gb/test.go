package main

import (
	"fmt"
	"go/build"
	"path/filepath"
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
		var pkgs []*gb.Package
		/**
		if A {
			var err error
			args, err = proj.SrcDirs[0].FindAll()
			if err != nil {
				return fmt.Errorf("could not fetch packages in srcpath %v: %v", proj.SrcDirs[0], err)
			}
		}
		*/
		for _, arg := range args {
			if arg == "." {
				var err error
				arg, err = filepath.Rel(ctx.Srcdirs()[0], mustGetwd())
				if err != nil {
					return err
				}
			}
			pkg := ctx.ResolvePackage(arg)
			if err := pkg.Result(); err != nil {
				if _, ok := err.(*build.NoGoError); ok {
					gb.Debugf("skipping %q", arg)
					continue
				}
				return fmt.Errorf("failed to resolve package %q: %v", arg, err)
			}
			pkgs = append(pkgs, pkg)
		}
		results := make(chan gb.Target, len(pkgs))
		go func() {
			defer close(results)
			for _, pkg := range pkgs {
				results <- gb.Test(pkg)
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
