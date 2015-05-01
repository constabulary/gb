package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
)

type Command struct {
	ShortDesc string
	Run       func(ctx *gb.Context, args []string) error
	AddFlags  func(fs *flag.FlagSet)
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

var commands = make(map[string]*Command)

// registerCommand registers a command for main.
// registerCommand should only be called from init().
func registerCommand(name string, command *Command) {
	commands[name] = command
}

func main() {
	args := os.Args
	if len(args) < 2 || args[1] == "-h" {
		fs.Usage()
		os.Exit(1)
	}

	gopath := filepath.SplitList(os.Getenv("GOPATH"))
	root, err := cmd.FindProjectroot(mustGetwd(), gopath)
	if err != nil {
		gb.Fatalf("could not locate project root: %v", err)
	}
	gb.Infof("project root %q", root)

	name := args[1]
	cmd, ok := commands[name]
	if !ok {
		if _, err := lookupPlugin(name); err != nil {
			gb.Errorf("unknown command %q", name)
			fs.Usage()
			os.Exit(1)
		}
		cmd = commands["plugin"]
		args = append([]string{"plugin"}, args...)
	}

	// add extra flags if necessary
	if cmd.AddFlags != nil {
		cmd.AddFlags(fs)
	}

	if err := fs.Parse(args[2:]); err != nil {
		gb.Fatalf("could not parse flags: %v", err)
	}

	// must be below fs.Parse because the -q and -v flags will log.Infof
	project := gb.NewProject(root)
	tc, err := gb.NewGcToolchain(*goroot, *goos, *goarch)
	if err != nil {
		gb.Fatalf("unable to construct toolchain: %v", err)
	}
	ctx := project.NewContext(tc)

	args = importPaths(ctx, fs.Args())
	gb.Debugf("args: %v", args)
	if err := cmd.Run(ctx, args); err != nil {
		gb.Fatalf("command %q failed: %v", name, err)
	}
}

// importPathsNoDotExpansion returns the import paths to use for the given
// command line, but it does no ... expansion.
func importPathsNoDotExpansion(ctx *gb.Context, args []string) []string {
	cwd, _ := os.Getwd()
	srcdir, _ := filepath.Rel(ctx.Srcdirs()[0], cwd)
	if srcdir == ".." {
		srcdir = "."
	}
	if len(args) == 0 {
		args = []string{"."}
	}
	var out []string
	for _, a := range args {
		// Arguments are supposed to be import paths, but
		// as a courtesy to Windows developers, rewrite \ to /
		// in command-line arguments.  Handles .\... and so on.
		if filepath.Separator == '\\' {
			a = strings.Replace(a, `\`, `/`, -1)
		}

		if a == "all" || a == "std" {
			out = append(out, ctx.AllPackages(a)...)
			continue
		}
		a = path.Join(srcdir, path.Clean(a))
		out = append(out, a)
	}
	return out
}

// importPaths returns the import paths to use for the given command line.
func importPaths(ctx *gb.Context, args []string) []string {
	args = importPathsNoDotExpansion(ctx, args)
	var out []string
	for _, a := range args {
		if strings.Contains(a, "...") {
			out = append(out, ctx.AllPackages(a)...)
			continue
		}
		out = append(out, a)
	}
	return out
}
