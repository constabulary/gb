package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
	"github.com/constabulary/gb/log"
)

var (
	fs   = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	cwd  string
	args []string
)

const (
	// enable to keep working directory
	noDestroyContext = false
)

func init() {
	fs.BoolVar(&log.Quiet, "q", log.Quiet, "suppress log messages below ERROR level")
	fs.BoolVar(&log.Verbose, "v", log.Verbose, "enable log levels below INFO level")
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
		plugin, err := lookupPlugin(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "FATAL: unknown command %q\n", name)
			fs.Usage()
			os.Exit(1)
		}
		command = &cmd.Command{
			Run: func(ctx *gb.Context, args []string) error {
				if len(args) < 1 {
					return fmt.Errorf("plugin: no command supplied")
				}
				args = append([]string{plugin}, args...)

				env := cmd.MergeEnv(os.Environ(), map[string]string{
					"GB_PROJECT_DIR": ctx.Projectdir(),
				})

				cmd := exec.Cmd{
					Path: plugin,
					Args: args,
					Env:  env,

					Stdin:  os.Stdin,
					Stdout: os.Stdout,
					Stderr: os.Stderr,
				}

				return cmd.Run()
			},
			// plugin should not interpret arguments
			ParseArgs: func(_ *gb.Context, _ string, args []string) []string { return args },
		}
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
		log.Fatalf("could not parse flags: %v", err)
	}

	args = fs.Args() // reset args to the leftovers from fs.Parse
	if command == commands["plugin"] {
		args = append([]string{name}, args...)
	}
	cwd, err := filepath.Abs(cwd) // if cwd was passed in via -R, make sure it is absolute
	if err != nil {
		log.Fatalf("could not make project root absolute: %v", err)
	}

	ctx, err := cmd.NewContext(
		cwd, // project root
		gb.GcToolchain(),
		gb.Gcflags(gcflags...),
		gb.Ldflags(ldflags...),
		gb.Tags(buildtags...),
	)
	if err != nil {
		log.Fatalf("unable to construct context: %v", err)
	}

	if !noDestroyContext {
		defer ctx.Destroy()
	}

	if command.ParseArgs != nil {
		args = command.ParseArgs(ctx, ctx.Projectdir(), args)
	} else {
		args = cmd.ImportPaths(ctx, cwd, args)
	}

	log.Debugf("args: %v", args)
	if err := command.Run(ctx, args); err != nil {
		if !noDestroyContext {
			ctx.Destroy()
		}
		log.Fatalf("command %q failed: %v", name, err)
	}
}
