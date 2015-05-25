package main

import (
	"fmt"
	"path/filepath"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
	"github.com/constabulary/gb/cmd/gb-vendor/vendor"
)

func init() {
	registerCommand("delete", DeleteCmd)
}

var DeleteCmd = &cmd.Command{
	ShortDesc: "deletes a local dependency",
	Run: func(ctx *gb.Context, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("delete: import path missing")
		}
		path := args[0]

		m, err := vendor.ReadManifest(manifestFile(ctx))
		if err != nil {
			return fmt.Errorf("could not load manifest: %T %v", err, err)
		}

		d, err := m.GetDependencyForImportpath(path)
		if err != nil {
			return fmt.Errorf("could not get dependency: %T %v", err, err)
		}

		err = m.RemoveDependency(d)
		if err != nil {
			return fmt.Errorf("dependency could not be deleted: %T %v", err, err)
		}

		localClone := vendor.GitClone{
			Path: filepath.Join(ctx.Projectdir(), "vendor", "src", path),
		}
		err = localClone.Destroy()
		if err != nil {
			return fmt.Errorf("dependency could not be deleted: %T %v", err, err)
		}

		if err := vendor.WriteManifest(manifestFile(ctx), m); err != nil {
			return err
		}

		return nil
	},
}
