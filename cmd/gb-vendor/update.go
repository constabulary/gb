package main

import (
	"flag"
	"fmt"
	"path/filepath"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
	"github.com/constabulary/gb/cmd/gb-vendor/vendor"
)

var (
	// gb vendor update flags

	// update all dependencies
	updateAll bool
)

func init() {
	registerCommand("update", UpdateCmd)
}

func addUpdateFlags(fs *flag.FlagSet) {
	fs.BoolVar(&updateAll, "all", false, "update all dependencies")
}

var UpdateCmd = &cmd.Command{
	ShortDesc: "updates a local dependency",
	Run: func(ctx *gb.Context, args []string) error {
		if len(args) != 1 && !updateAll {
			return fmt.Errorf("update: import path or --all flag is missing")
		} else if len(args) == 1 && updateAll {
			return fmt.Errorf("update: you cannot specify path and --all flag at once")
		}

		m, err := vendor.ReadManifest(manifestFile(ctx))
		if err != nil {
			return fmt.Errorf("could not load manifest: %v", err)
		}

		var dependencies []vendor.Dependency
		if updateAll {
			dependencies = make([]vendor.Dependency, len(m.Dependencies))
			copy(dependencies, m.Dependencies)
		} else {
			p := args[0]
			dependency, err := m.GetDependencyForImportpath(p)
			if err != nil {
				return fmt.Errorf("could not get dependency: %v", err)
			}
			dependencies = append(dependencies, dependency)
		}

		for _, d := range dependencies {
			url := d.Repository
			path := d.Importpath

			err = m.RemoveDependency(d)
			if err != nil {
				return fmt.Errorf("dependency could not be deleted from manifest: %v", err)
			}

			localClone := vendor.GitClone{
				Path: filepath.Join(ctx.Projectdir(), "vendor", "src", path),
			}
			err = localClone.Destroy()
			if err != nil {
				return fmt.Errorf("dependency could not be deleted: %v", err)
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

			if err := copypath(dst, src); err != nil {
				return err
			}

			if err := wc.Destroy(); err != nil {
				return err
			}
			fmt.Println(dependencies)
		}

		return vendor.WriteManifest(manifestFile(ctx), m)
	},
	AddFlags: addUpdateFlags,
}
