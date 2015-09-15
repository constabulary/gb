// Package command holds support functions and types for writing gb and gb plugins
package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/log"
)

// Command represents a subcommand, or plugin that is executed within
// a gb project.
type Command struct {
	// Name of the command
	Name string

	// UsageLine demonstrates how to use this command
	UsageLine string

	// Single line description of the purpose of the command
	Short string

	// Description of this command
	Long string

	// Run is invoked with a Context derived from the Project and arguments
	// left over after flag parsing.
	Run func(ctx *gb.Context, args []string) error

	// AddFlags installs additional flags to be parsed before Run.
	AddFlags func(fs *flag.FlagSet)

	// Allow plugins to modify arguments
	FlagParse func(fs *flag.FlagSet, args []string) error

	// ParseArgs provides an alternative method to parse arguments.
	// By default, arguments will be parsed as import paths with
	// ImportPaths
	ParseArgs func(ctx *gb.Context, cwd string, args []string) []string
}

// Runnable indicates this is a command that can be involved.
// Non runnable commands are only informational.
func (c *Command) Runnable() bool { return c.Run != nil }

// Hidden indicates this is a command which is hidden from help / alldoc.go.
func (c *Command) Hidden() bool { return c.Name == "depset" }

// RunCommand detects the project root, parses flags and runs the Command.
func RunCommand(fs *flag.FlagSet, cmd *Command, projectroot, goroot string, args []string) error {
	if cmd.AddFlags != nil {
		cmd.AddFlags(fs)
	}
	if err := fs.Parse(args); err != nil {
		fs.Usage()
		os.Exit(1)
	}
	args = fs.Args() // reset to the remaining arguments

	ctx, err := NewContext(projectroot, gb.GcToolchain())
	if err != nil {
		return fmt.Errorf("unable to construct context: %v", err)
	}
	defer ctx.Destroy()

	log.Debugf("args: %v", args)
	return cmd.Run(ctx, args)
}

// NewContext creates a gb.Context for the project root.
func NewContext(projectroot string, options ...func(*gb.Context) error) (*gb.Context, error) {
	if projectroot == "" {
		return nil, fmt.Errorf("project root is blank")
	}

	root, err := FindProjectroot(projectroot)
	if err != nil {
		return nil, fmt.Errorf("could not locate project root: %v", err)
	}
	project := gb.NewProject(root,
		gb.SourceDir(filepath.Join(root, "src")),
		gb.SourceDir(filepath.Join(root, "vendor", "src")),
	)

	log.Debugf("project root %q", project.Projectdir())
	return project.NewContext(options...)
}

func mkdir(path string) error {
	return os.MkdirAll(path, 0755)
}
