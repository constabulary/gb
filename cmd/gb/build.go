package main

import (
	"flag"
	"time"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
)

func init() {
	registerCommand(BuildCmd)
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

	ldflags string
)

func addBuildFlags(fs *flag.FlagSet) {
	// TODO(dfc) this should accept a *gb.Context
	fs.BoolVar(&A, "a", false, "build all packages in this project")
	fs.BoolVar(&R, "r", false, "perform a release build")
	fs.BoolVar(&F, "f", false, "rebuild up to date packages")
	fs.BoolVar(&FF, "F", false, "do not cache built packages")
	fs.StringVar(&ldflags, "ldflags", "", "flags passed to the linker")
}

var BuildCmd = &cmd.Command{
	Name:      "build",
	Short:     "build a package",
	UsageLine: "build [build flags] [packages]",
	Long: `Build compiles the packages named by the import paths, along with their dependencies.

The build flags are 

	-f
		ignore cached packages if present, new packages built will overwrite any cached packages.
		This effectively disables incremental compilation.
	-F
		do not cache packages, cached packages will still be used for incremental compilation.
		-f -F is advised to disable the package caching system.
	-q
		decreases verbosity, effectively raising the output level to ERROR.
		In a successful build, no output will be displayed.
	-R
		sets the base of the project root search path from the current working directory to the value supplied.
		Effectively gb changes working directory to this path before searching for the project root.
	-v
		increases verbosity, effectively lowering the output level from INFO to DEBUG.
	-ldflags 'flag list'
		arguments to pass on each linker invocation.

The list flags accept a space-separated list of strings. To embed spaces in an element in the list, surround it with either single or double quotes.

For more about specifying packages, see 'gb help packages'. For more about where packages and binaries are installed, run 'gb help project'.`,
	Run: func(ctx *gb.Context, args []string) error {
		// TODO(dfc) run should take a *gb.Context not a *gb.Project
		t0 := time.Now()
		ctx.Force = F
		ctx.SkipInstall = FF
		defer func() {
			gb.Debugf("build duration: %v %v", time.Since(t0), ctx.Statistics.String())
		}()

		pkgs, err := cmd.ResolvePackages(ctx, args...)
		if err != nil {
			ctx.Destroy()
			return err
		}
		if err := gb.Build(pkgs...); err != nil {
			ctx.Destroy()
			return err
		}
		return ctx.Destroy()
	},
	AddFlags: addBuildFlags,
}
