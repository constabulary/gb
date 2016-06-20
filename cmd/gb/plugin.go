package main

import (
	"os/exec"

	"github.com/constabulary/gb/cmd"
	"github.com/pkg/errors"
)

func init() {
	registerCommand(PluginCmd)
}

var PluginCmd = &cmd.Command{
	Name:  "plugin",
	Short: "plugin information",
	Long: `gb supports git style plugins.

A gb plugin is anything in the $PATH with the prefix gb-. In other words
gb-something, becomes gb something.

gb plugins are executed from the parent gb process with the environment
variable, GB_PROJECT_DIR set to the root of the current project.

gb plugins can be executed directly but this is rarely useful, so authors
should attempt to diagnose this by looking for the presence of the
GB_PROJECT_DIR environment key.
`,
}

func lookupPlugin(arg string) (string, error) {
	plugin := "gb-" + arg
	path, err := exec.LookPath(plugin)
	return path, errors.Wrapf(err, "plugin: unable to locate %q", plugin)
}
