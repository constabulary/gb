package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
)

func init() {
	registerCommand("plugin", PluginCmd)
}

var PluginCmd = &Command{
	Run: func(proj *gb.Project, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("plugin: no command supplied")
		}

		plugin := "gb-" + args[0]
		args[0] = plugin

		path, err := exec.LookPath(plugin)
		if err != nil {
			return fmt.Errorf("plugin: unable to locate %q: %v", plugin, err)
		}

		env := cmd.MergeEnv(os.Environ(), map[string]string{
			"GB_PROJECT_DIR": proj.Projectdir(),
		})

		cmd := exec.Cmd{
			Path: path,
			Args: args,
			Env:  env,

			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}

		return cmd.Run()
	},
}
