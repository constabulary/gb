package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
)

var (
	fs          = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	projectroot = os.Getenv("GB_PROJECT_DIR")
	args        []string
)

func init() {
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
	root, err := cmd.FindProjectroot(projectroot)
	if err != nil {
		gb.Fatalf("could not locate project root: %v", err)
	}
	project := gb.NewProject(root)
	gb.Debugf("project root %q", project.Projectdir())

	args := os.Args
	if len(args) < 2 || args[1] == "-h" {
		fs.Usage()
		os.Exit(1)
	}

	name := args[1]
	command, ok := commands[name]
	if !ok {
		fs.Usage()
		os.Exit(1)
	}

	// add extra flags if necessary
	if command.AddFlags != nil {
		command.AddFlags(fs)
	}

	if command.FlagParse != nil {
		err = command.FlagParse(fs, args)
	} else {
		err = fs.Parse(args[2:])
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
		gb.Fatalf("command %q failed: %v", name, err)
	}
}

const manifestfile = "manifest"

func manifestFile(ctx *gb.Context) string {
	return filepath.Join(ctx.Projectdir(), "vendor", manifestfile)
}

// copypath copies the contents of src to dst, excluding any file or
// directory that starts with a period.
func copypath(dst string, src string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasPrefix(filepath.Base(path), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}

		dst := filepath.Join(dst, path[len(src):])
		return copyfile(dst, path)
	})
}

func copyfile(dst, src string) error {
	err := mkdir(filepath.Dir(dst))
	if err != nil {
		return fmt.Errorf("copyfile: mkdirall: %v", err)
	}
	r, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("copyfile: open(%q): %v", src, err)
	}
	defer r.Close()
	w, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("copyfile: create(%q): %v", dst, err)
	}
	fmt.Printf("copyfile(dst: %v, src: %v)\n", dst, src)
	_, err = io.Copy(w, r)
	return err
}

func mkdir(path string) error {
	return os.MkdirAll(path, 0755)
}
