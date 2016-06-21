package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
	"github.com/constabulary/gb/internal/debug"
)

var (
	fs          = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	projectroot = os.Getenv("GB_PROJECT_DIR")
)

func init() {
	fs.Usage = usage
}

var commands = []*cmd.Command{
	cmdFetch,
	cmdUpdate,
	cmdList,
	cmdDelete,
	cmdPurge,
	cmdRestore,
}

func main() {
	fatalf := func(format string, args ...interface{}) {
		fmt.Fprintf(os.Stderr, "FATAL: "+format+"\n", args...)
		os.Exit(1)
	}

	args := os.Args[1:]

	switch {
	case len(args) < 1, args[0] == "-h", args[0] == "-help":
		printUsage(os.Stdout)
		os.Exit(0)
	case args[0] == "help":
		help(args[1:])
		return
	case projectroot == "":
		fatalf("don't run this binary directly, it is meant to be run as 'gb vendor ...'")
	default:
	}

	root, err := cmd.FindProjectroot(projectroot)
	if err != nil {
		fatalf("could not locate project root: %v", err)
	}
	project := gb.NewProject(root)

	debug.Debugf("project root %q", project.Projectdir())

	for _, command := range commands {
		if command.Name == args[0] && command.Runnable() {

			// add extra flags if necessary
			if command.AddFlags != nil {
				command.AddFlags(fs)
			}

			if command.FlagParse != nil {
				err = command.FlagParse(fs, args)
			} else {
				err = fs.Parse(args[1:])
			}
			if err != nil {
				fatalf("could not parse flags: %v", err)
			}
			args = fs.Args() // reset args to the leftovers from fs.Parse
			debug.Debugf("args: %v", args)

			ctx, err := gb.NewContext(project, gb.GcToolchain())
			if err != nil {
				fatalf("unable to construct context: %v", err)
			}
			defer ctx.Destroy()

			if err := command.Run(ctx, args); err != nil {
				fatalf("command %q failed: %v", command.Name, err)
			}
			return
		}
	}
	fatalf("unknown command %q ", args[0])
}

const manifestfile = "manifest"

func manifestFile(ctx *gb.Context) string {
	return filepath.Join(ctx.Projectdir(), "vendor", manifestfile)
}
