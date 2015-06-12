package main

import (
	"fmt"
	"os"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
	"github.com/pkg/exec"
)

func init() {
	registerCommand(PluginCmd)
}

var PluginCmd = &cmd.Command{
	Name:      "plugin",
	UsageLine: `plugin command`,
	Short:     "run a plugin",
	Long: `gb supports git style plugins

See gb help plugins.

`,
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

		cmd := exec.Command(args[0], args[1:]...)
		cmd.Cmd.Env = env
		return cmd.Run(
			exec.Stdin(os.Stdin),
			exec.Stdout(os.Stdout),
			exec.Stderr(os.Stderr),
		)
	},
	// plugin should not interpret arguments
	ParseArgs: func(_ *gb.Context, _ string, args []string) []string { return args },
}

func lookupPlugin(arg string) (string, error) {
	plugin := "gb-" + arg
	path, err := exec.LookPath(plugin)
	if err != nil {
		return "", fmt.Errorf("plugin: unable to locate %q: %v", plugin, err)
	}
	return path, nil
}
