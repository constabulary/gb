package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
)

func init() {
	registerCommand(PluginCmd)
}

var PluginCmd = &cmd.Command{
	Name:      "plugin",
	ShortDesc: "run a plugin",
	Run: func(ctx *gb.Context, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("plugin: no command supplied")
		}

		path, err := lookupPlugin(args[0])
		if err != nil {
			return err
		}
		args[0] = path

		env := cmd.MergeEnv(os.Environ(), map[string]string{
			"GB_PROJECT_DIR": ctx.Projectdir(),
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

func lookupPlugin(arg string) (string, error) {
	plugin := "gb-" + arg
	path, err := exec.LookPath(plugin)
	if err != nil {
		return "", fmt.Errorf("plugin: unable to locate %q: %v", plugin, err)
	}
	return path, nil
}
