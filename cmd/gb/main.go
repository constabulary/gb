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
		fatalf("could not parse flags: %v", err)
	}

	args = fs.Args() // reset args to the leftovers from fs.Parse

	debug.Debugf("args: %v", args)

	if command == commands["plugin"] {
		args = append([]string{name}, args...)
	}
	cwd, err := filepath.Abs(cwd) // if cwd was passed in via -R, make sure it is absolute
	if err != nil {
		fatalf("could not make project root absolute: %v", err)
	}

	ctx, err := cmd.NewContext(
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

	if err != nil {
		fatalf("unable to construct context: %v", err)
	}

	if !command.SkipParseArgs {
		args = importPaths(ctx, cwd, args)
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
