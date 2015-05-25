package main

import (
	"fmt"
	"path/filepath"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
	"github.com/constabulary/gb/cmd/gb-vendor/vendor"
)

func init() {
	registerCommand("update", UpdateCmd)
}

var UpdateCmd = &cmd.Command{
	ShortDesc: "updates a local dependency",
	Run: func(ctx *gb.Context, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("update: import path missing")
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

		url := d.Repository
		err = m.RemoveDependency(d)
		if err != nil {
			return fmt.Errorf("dependency could not be deleted from manifest: %T %v", err, err)
		}

		localClone := vendor.GitClone{
			Path: filepath.Join(ctx.Projectdir(), "vendor", "src", path),
		}
		err = localClone.Destroy()
		if err != nil {
			return fmt.Errorf("dependency could not be deleted: %T %v", err, err)
		}

		repo := vendor.GitRepo{
			URL: url,
		}

		wc, err := repo.Clone()
		if err != nil {
			return err
		}

		rev, err := wc.Revision()
		if err != nil {
			return err
		}

		branch, err := wc.Branch()
		if err != nil {
			return err
		}

		dep := vendor.Dependency{
			Importpath: path,
			Repository: url,
			Revision:   rev,
			Branch:     branch,
			Path:       "",
		}

		if err := m.AddDependency(dep); err != nil {
			return err
		}

		dst := filepath.Join(ctx.Projectdir(), "vendor", "src", dep.Importpath)
		src := filepath.Join(wc.Dir(), dep.Path)

		if err := copypath(dst, src, ".git"); err != nil {
			return err
		}

		if err := vendor.WriteManifest(manifestFile(ctx), m); err != nil {
			return err
		}
		return wc.Destroy()
	},
}
