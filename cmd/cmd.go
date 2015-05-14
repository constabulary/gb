// Package command holds support functions and types for writing gb and gb plugins
package cmd

import (
	"flag"

	"github.com/constabulary/gb"
)

// Command represents a subcommand, or plugin that is executed within
// a gb project.
type Command struct {
	// Single line description of the purpose of the command
	ShortDesc string

	// Run is invoked with a Context derived from the Project and arguments
	// left over after flag parsing.
	Run func(ctx *gb.Context, args []string) error

	// AddFlags installs additional flags to be parsed before Run.
	AddFlags func(fs *flag.FlagSet)
}
