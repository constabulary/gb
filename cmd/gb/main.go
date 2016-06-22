package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
	"github.com/constabulary/gb/cmd/gb/internal/match"
	"github.com/constabulary/gb/internal/debug"
)

var (
	fs  = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	cwd string
)

const (
	// disable to keep working directory
	destroyContext = true
)

func init() {
	fs.StringVar(&cwd, "R", cmd.MustGetwd(), "set the project root") // actually the working directory to start the project root search
	fs.Usage = usage
}

var commands = make(map[string]*cmd.Command)

// registerCommand registers a command for main.
// registerCommand should only be called from init().
func registerCommand(command *cmd.Command) {
	setCommandDefaults(command)
	commands[command.Name] = command
}

// atExit functions are called in sequence at the exit of the program.
var atExit []func() error

// exit runs all atExit functions, then calls os.Exit(code).
func exit(code int) {
	for _, fn := range atExit {
		fn()
	}
	os.Exit(code)
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "FATAL: "+format+"\n", args...)
	exit(1)
}

func main() {
	args := os.Args
	if len(args) < 2 || args[1] == "-h" {
		fs.Usage() // usage calles exit(2)
	}
	name := args[1]
	if name == "help" {
		help(args[2:])
		exit(0)
	}

	command := lookupCommand(name)

	// add extra flags if necessary
	command.AddFlags(fs)

	// parse 'em
	err := command.FlagParse(fs, args)
	if err != nil {
		fatalf("could not parse flags: %v", err)
	}

	// reset args to the leftovers from fs.Parse
	args = fs.Args()

	// if this is the plugin command, ensure the name of the
	// plugin is first in the list of arguments.
	if command == commands["plugin"] {
		args = append([]string{name}, args...)
	}

	// if cwd was passed in via -R, make sure it is absolute
	cwd, err := filepath.Abs(cwd)
	if err != nil {
		fatalf("could not make project root absolute: %v", err)
	}

	// construct a project context at the current working directory.
	ctx, err := newContext(cwd)
	if err != nil {
		fatalf("unable to construct context: %v", err)
	}

	// unless the command wants to handle its own arguments, process
	// arguments into import paths.
	if !command.SkipParseArgs {
		srcdir := filepath.Join(ctx.Projectdir(), "src")
		for _, a := range args {
			// support the "all" build alias. This used to be handled
			// in match.ImportPaths, but that's too messy, so if "all"
			// is present in the args, replace it with "..." and set cwd
			// to srcdir.
			if a == "all" {
				args = []string{"..."}
				cwd = srcdir
				break
			}
		}
		args = match.ImportPaths(srcdir, cwd, args)
	}

	debug.Debugf("args: %v", args)

	if destroyContext {
		atExit = append(atExit, ctx.Destroy)
	}

	if err := command.Run(ctx, args); err != nil {
		fatalf("command %q failed: %v", name, err)
	}
	exit(0)
}

func lookupCommand(name string) *cmd.Command {
	command, ok := commands[name]
	if (command != nil && !command.Runnable()) || !ok {
		plugin, err := lookupPlugin(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "FATAL: unknown command %q\n", name)
			fs.Usage() // usage calles exit(2)
		}
		command = &cmd.Command{
			Run: func(ctx *gb.Context, args []string) error {
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
			SkipParseArgs: true,
		}
	}
	setCommandDefaults(command)
	return command
}

func setCommandDefaults(command *cmd.Command) {

	// add a dummy default AddFlags field if none provided.
	if command.AddFlags == nil {
		command.AddFlags = func(*flag.FlagSet) {}
	}

	// add the default flag parsing if not overrriden.
	if command.FlagParse == nil {
		command.FlagParse = func(fs *flag.FlagSet, args []string) error {
			return fs.Parse(args[2:])
		}
	}
}

func newContext(cwd string) (*gb.Context, error) {
	return cmd.NewContext(
		cwd, // project root
		gb.GcToolchain(),
		gb.Gcflags(gcflags...),
		gb.Ldflags(ldflags...),
		gb.Tags(buildtags...),
		func(c *gb.Context) error {
			if !race {
				return nil
			}

			// check this is a supported platform
			if runtime.GOARCH != "amd64" {
				fatalf("race detector not supported on %s/%s", runtime.GOOS, runtime.GOARCH)
			}
			switch runtime.GOOS {
			case "linux", "windows", "darwin", "freebsd":
				// supported
			default:
				fatalf("race detector not supported on %s/%s", runtime.GOOS, runtime.GOARCH)
			}

			// check the race runtime is built
			_, err := os.Stat(filepath.Join(runtime.GOROOT(), "pkg", fmt.Sprintf("%s_%s_race", runtime.GOOS, runtime.GOARCH), "runtime.a"))
			if os.IsNotExist(err) || err != nil {
				fatalf("go installation at %s is missing race support. See https://getgb.io/faq/#missing-race-support", runtime.GOROOT())
			}

			return gb.WithRace(c)
		},
	)
}
