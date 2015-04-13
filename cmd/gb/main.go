package main

import (
	"flag"
	"os"
	"path/filepath"
	"runtime"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
)

type Command struct {
	Run      func(project *gb.Project, args []string) error
	AddFlags func(fs *flag.FlagSet)
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		gb.Fatalf("unable to determine current working directory: %v", err)
	}
	return wd
}

var (
	fs        = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	goos      = fs.String("goos", runtime.GOOS, "override GOOS")
	goarch    = fs.String("goarch", runtime.GOARCH, "override GOARCH")
	goroot    = fs.String("goroot", runtime.GOROOT(), "override GOROOT")
	toolchain = fs.String("toolchain", "gc", "choose go compiler toolchain")
)

func init() {
	fs.BoolVar(&gb.Quiet, "q", gb.Quiet, "suppress log messages below ERROR level")
	fs.BoolVar(&gb.Verbose, "v", gb.Verbose, "enable log levels below INFO level")
}

var commands = make(map[string]*Command)

// registerCommand registers a command for main.
// registerCommand should only be called from init().
func registerCommand(name string, command *Command) {
	commands[name] = command
}

func main() {
	args := os.Args
	if len(args) < 2 {
		gb.Fatalf("no command supplied")
	}

	gopath := filepath.SplitList(os.Getenv("GOPATH"))
	root, err := cmd.FindProjectroot(mustGetwd(), gopath)
	if err != nil {
		gb.Fatalf("could not locate project root: %v", err)
	}

	project := gb.NewProject(root)

	name := args[1]
	cmd, ok := commands[name]
	if !ok {
		gb.Errorf("unknown command %q", args[1])
		fs.PrintDefaults()
		os.Exit(1)
	}

	// add extra flags if necessary
	if cmd.AddFlags != nil {
		cmd.AddFlags(fs)
	}

	if err := fs.Parse(args[2:]); err != nil {
		gb.Fatalf("could not parse flags: %v", err)
	}

	// must be below fs.Parse because the -q and -v flags will log.Infof
	gb.Infof("project root %q", root)
	args = fs.Args()
	if len(args) == 0 {
		args = []string{"."}
	}
	if err := cmd.Run(project, args); err != nil {
		gb.Fatalf("command %q failed: %v", name, err)
	}
}
