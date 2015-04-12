package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/constabulary/gb"
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
		args = args[1:]

		path, err := exec.LookPath(plugin)
		if err != nil {
			return fmt.Errorf("plugin: unable to locate %q: %v", plugin, err)
		}

		env := mergeEnv(os.Environ(), map[string]string{
			"GB_PROJECT_DIR": proj.Projectdir(),
		})

		cmd := exec.Cmd{
			Path: path,
			Args: args,
			Env: env,

			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}

		return cmd.Run()
	},
}

func mergeEnv(env []string, args map[string]string) []string {
	for k, v := range args {
		env = append(env, fmt.Sprintf("%s=%q", k, v))
	}
	return env
}
