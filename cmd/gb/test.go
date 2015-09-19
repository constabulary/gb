package main

import (
	"flag"
	"fmt"
	"os"

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
Test automates testing the packages named by the import paths.

'gb test' recompiles each package along with any files with names matching
the file pattern "*_test.go".

See 'go help test'.
`,
	Run: func(ctx *gb.Context, args []string) error {
		ctx.Force = F
		ctx.SkipInstall = FF
		pkgs, err := cmd.ResolvePackagesWithTests(ctx, args...)
		if err != nil {
			return err
		}

		test, err := cmd.TestPackages(cmd.TestFlags(tfs), pkgs...)
		if err != nil {
			return err
		}

		if dotfile != "" {
			f, err := os.Create(dotfile)
			if err != nil {
				return err
			}
			defer f.Close()
			printActions(f, test)
		}

		return gb.ExecuteConcurrent(test, P)
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
