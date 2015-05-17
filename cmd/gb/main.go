package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
)

var (
	fs     = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	goroot = fs.String("goroot", runtime.GOROOT(), "override GOROOT")
	cwd    string
)

func init() {
	fs.BoolVar(&gb.Quiet, "q", gb.Quiet, "suppress log messages below ERROR level")
	fs.BoolVar(&gb.Verbose, "v", gb.Verbose, "enable log levels below INFO level")
	fs.StringVar(&cwd, "R", cmd.MustGetwd(), "set the project root") // actually the working directory to start the project root search

	// TODO some flags are specific to a specific commands
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage:")
		for name, cmd := range commands {
			fmt.Fprintf(os.Stderr, "  gb %s [flags] [package] - %s\n",
				name, cmd.ShortDesc)
		}
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Flags:")
		fs.PrintDefaults()
	}
}

var commands = make(map[string]*cmd.Command)

// registerCommand registers a command for main.
// registerCommand should only be called from init().
func registerCommand(name string, command *cmd.Command) {
	commands[name] = command
}

func main() {
	args := os.Args
	if len(args) < 2 || args[1] == "-h" {
		fs.Usage()
		os.Exit(1)
	}

	name := args[1]
	parseargs := name != "plugin"
	command, ok := commands[name]
	if !ok {
		if _, err := lookupPlugin(name); err != nil {
			gb.Errorf("unknown command %q", name)
			fs.Usage()
			os.Exit(1)
		}
		command = commands["plugin"]
		args = append([]string{"plugin"}, args...)
		parseargs = false // don't parse args as import paths
	}

	// add extra flags if necessary
	if command.AddFlags != nil {
		command.AddFlags(fs)
	}

	if err := fs.Parse(args[2:]); err != nil {
		gb.Fatalf("could not parse flags: %v", err)
	}
	args = fs.Args()              // reset args to the leftovers from fs.Parse
	cwd, err := filepath.Abs(cwd) // if cwd was passed in via -R, make sure it is absolute
	if err != nil {
		gb.Fatalf("could not make project root absolute: %v", err)
	}

	root, err := cmd.FindProjectroot(cwd)
	if err != nil {
		gb.Fatalf("could not locate project root: %v", err)
	}
	project := gb.NewProject(root)

	gb.Debugf("project root %q", project.Projectdir())

	ctx, err := project.NewContext(
		gb.GcToolchain(gb.Goroot(*goroot)),
		gb.Ldflags(ldflags),
	)
	if err != nil {
		gb.Fatalf("unable to construct context: %v", err)
	}

	if parseargs {
		args = cmd.ImportPaths(ctx, cwd, args)
	}
	gb.Debugf("args: %v", args)
	if err := command.Run(ctx, args); err != nil {
		gb.Fatalf("command %q failed: %v", name, err)
	}
}
