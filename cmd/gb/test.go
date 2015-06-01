package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
)

func init() {
	registerCommand(TestCmd)
}

var (
	tfs           []string // Arguments passed to the test binary
	testProfile   bool
	testCover     bool
	testCoverMode string
	testCoverPkg  string
)

func addTestFlags(fs *flag.FlagSet) {
	addBuildFlags(fs)
	fs.BoolVar(&testCover, "cover", false, "enable coverage analysis")
	fs.StringVar(&testCoverMode, "covermode", "set", "Set covermode: set (default), count, atomic")
	fs.StringVar(&testCoverPkg, "coverpkg", "", "enable coverage analysis")
}

var TestCmd = &cmd.Command{
	Name:      "test",
	UsageLine: "test [build flags] [packages] [flags for test binary]",
	Short:     "test packages",
	Long: `
'gb test' automates testing the packages named by the import paths.

'gb test' recompiles each package along with any files with names matching
the file pattern "*_test.go".

See 'go help test'

`,
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
		if err := cmd.Test(cmd.TestFlags(tfs), pkgs...); err != nil {
			return err
		}
		return ctx.Destroy()
	},
	AddFlags: addTestFlags,
	FlagParse: func(flags *flag.FlagSet, args []string) error {
		var err error
		args, tfs, err = cmd.TestFlagsExtraParse(args[2:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "gb test: %s\n", err)
			fmt.Fprintf(os.Stderr, `run "go help test" or "go help testflag" for more information`+"\n")
			os.Exit(2)
		}
		return flags.Parse(args)
	},
}
