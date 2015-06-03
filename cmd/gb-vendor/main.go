package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
)

var (
	fs          = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	projectroot = os.Getenv("GB_PROJECT_DIR")
	args        []string
)

func init() {
	fs.Usage = func() {
		printUsage(os.Stderr)
		os.Exit(2)
	}
}

var commands = []*cmd.Command{
	cmdFetch,
	cmdUpdate,
	cmdList,
	cmdDelete,
	cmdPurge,
}

func main() {
	root, err := cmd.FindProjectroot(projectroot)
	if err != nil {
		gb.Fatalf("could not locate project root: %v", err)
	}
	project := gb.NewProject(root)
	gb.Debugf("project root %q", project.Projectdir())

	args := os.Args[1:]
	if len(args) < 1 || args[0] == "-h" {
		fs.Usage()
		os.Exit(1)
	}

	if args[0] == "help" {
		help(args[1:])
		return
	}

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
				gb.Fatalf("could not parse flags: %v", err)
			}
			args = fs.Args() // reset args to the leftovers from fs.Parse
			gb.Debugf("args: %v", args)

			ctx, err := project.NewContext(
				gb.GcToolchain(),
			)
			if err != nil {
				gb.Fatalf("unable to construct context: %v", err)
			}

			if err := command.Run(ctx, args); err != nil {
				gb.Fatalf("command %q failed: %v", command.Name, err)
			}
			return
		}
	}
	gb.Fatalf("unknown command %q ", args[0])
}

const manifestfile = "manifest"

func manifestFile(ctx *gb.Context) string {
	return filepath.Join(ctx.Projectdir(), "vendor", manifestfile)
}
