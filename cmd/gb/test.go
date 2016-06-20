package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
	"github.com/constabulary/gb/internal/debug"
	"github.com/constabulary/gb/test"
)

func init() {
	registerCommand(TestCmd)
}

var (
	tfs           []string // Arguments passed to the test binary
	testCover     bool
	testCoverMode string
	testCoverPkg  string
	testVerbose   bool // enable verbose output of test commands
	testNope      bool // do not execute test binaries, compile and link only
)

func addTestFlags(fs *flag.FlagSet) {
	addBuildFlags(fs)
	fs.BoolVar(&testCover, "cover", false, "enable coverage analysis")
	fs.StringVar(&testCoverMode, "covermode", "set", "Set covermode: set (default), count, atomic")
	fs.StringVar(&testCoverPkg, "coverpkg", "", "enable coverage analysis")
	fs.BoolVar(&testVerbose, "v", false, "enable verbose output of subcommands")
	fs.BoolVar(&testNope, "n", false, "do not execute test binaries, compile only")
}

var TestCmd = &cmd.Command{
	Name:      "test",
	UsageLine: "test [build flags] -n -v [packages] [flags for test binary]",
	Short:     "test packages",
	Long: `
Test automates testing the packages named by the import paths.

'gb test' recompiles each package along with any files with names matching
the file pattern "*_test.go".

Flags:

        -v
                print output from test subprocess.
	-n
		do not execute test binaries, compile only
`,
	Run: func(ctx *gb.Context, args []string) error {
		ctx.Force = F
		ctx.Install = !FF
		ctx.Verbose = testVerbose
		ctx.Nope = testNope
		r := test.TestResolver(ctx)

		// gb build builds packages in dependency order, however
		// gb test tests packages in alpha order. This matches the
		// expected behaviour from go test; tests are executed in
		// stable order.
		sort.Strings(args)

		pkgs, err := resolveRootPackages(r, args...)
		if err != nil {
			return err
		}

		test, err := test.TestPackages(TestFlags(tfs), pkgs...)
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

		startSigHandlers()
		return gb.ExecuteConcurrent(test, P, interrupted)
	},
	AddFlags: addTestFlags,
	FlagParse: func(flags *flag.FlagSet, args []string) error {
		var err error
		debug.Debugf("%s", args)
		args, tfs, err = TestFlagsExtraParse(args[2:])
		debug.Debugf("%s %s", args, tfs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gb test: %s\n", err)
			fmt.Fprintf(os.Stderr, `run "go help test" or "go help testflag" for more information`+"\n")
			exit(2)
		}
		return flags.Parse(args)
	},
}
