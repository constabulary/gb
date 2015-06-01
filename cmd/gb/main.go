package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
)

var (
	fs   = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	cwd  string
	args []string
)

func init() {
	fs.BoolVar(&gb.Quiet, "q", gb.Quiet, "suppress log messages below ERROR level")
	fs.BoolVar(&gb.Verbose, "v", gb.Verbose, "enable log levels below INFO level")
	fs.StringVar(&cwd, "R", cmd.MustGetwd(), "set the project root") // actually the working directory to start the project root search

	fs.Usage = usage
}

var commands = make(map[string]*cmd.Command)

// registerCommand registers a command for main.
// registerCommand should only be called from init().
func registerCommand(command *cmd.Command) {
	commands[command.Name] = command
}

func main() {
	args := os.Args
	if len(args) < 2 || args[1] == "-h" {
		fs.Usage()
		os.Exit(1)
	}
	name := args[1]
	if name == "help" {
		help(args[2:])
		return
	}
	command, ok := commands[name]
	if (command != nil && !command.Runnable()) || !ok {
		if _, err := lookupPlugin(name); err != nil {
			gb.Errorf("unknown command %q", name)
			fs.Usage()
			os.Exit(1)
		}
		command = commands["plugin"]
		args = append([]string{"plugin"}, args...)
	}

	// add extra flags if necessary
	if command.AddFlags != nil {
		command.AddFlags(fs)
	}

	var err error
	if command.FlagParse != nil {
		err = command.FlagParse(fs, args)
	} else {
		err = fs.Parse(args[2:])
	}
	if err != nil {
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
		gb.GcToolchain(),
		gb.Ldflags(ldflags),
	)
	if err != nil {
		gb.Fatalf("unable to construct context: %v", err)
	}

	if command.ParseArgs != nil {
		args = command.ParseArgs(ctx, root, args)
	} else {
		args = cmd.ImportPaths(ctx, root, args)
	}

	gb.Debugf("args: %v", args)
	if err := command.Run(ctx, args); err != nil {
		gb.Fatalf("command %q failed: %v", name, err)
	}
}
