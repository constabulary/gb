package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
	"github.com/constabulary/gb/log"
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
	cmdRestore,
}

func main() {
	args := os.Args[1:]

	switch {
	case len(args) < 1, args[0] == "-h", args[0] == "-help":
		fs.Usage()
		os.Exit(1)
	case args[0] == "help":
		help(args[1:])
		return
	case projectroot == "":
		log.Fatalf("don't run this binary directly, it is meant to be run as 'gb vendor ...'")
	default:
	}

	root, err := cmd.FindProjectroot(projectroot)
	if err != nil {
		log.Fatalf("could not locate project root: %v", err)
	}
	project := gb.NewProject(root,
		gb.SourceDir(filepath.Join(root, "src")),
		gb.SourceDir(filepath.Join(root, "vendor", "src")))
	log.Debugf("project root %q", project.Projectdir())

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
				log.Fatalf("could not parse flags: %v", err)
			}
			args = fs.Args() // reset args to the leftovers from fs.Parse
			log.Debugf("args: %v", args)

			ctx, err := project.NewContext(
				gb.GcToolchain(),
			)
			if err != nil {
				log.Fatalf("unable to construct context: %v", err)
			}
			defer ctx.Destroy()

			if err := command.Run(ctx, args); err != nil {
				log.Fatalf("command %q failed: %v", command.Name, err)
			}
			return
		}
	}
	log.Fatalf("unknown command %q ", args[0])
}

const manifestfile = "manifest"

func manifestFile(ctx *gb.Context) string {
	return filepath.Join(ctx.Projectdir(), "vendor", manifestfile)
}
