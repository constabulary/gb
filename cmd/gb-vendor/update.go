package main

import (
	"flag"
	"fmt"
	"path/filepath"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
	"github.com/constabulary/gb/vendor"
)

var (
	// gb vendor update flags

	// update all dependencies
	updateAll bool
)

func addUpdateFlags(fs *flag.FlagSet) {
	fs.BoolVar(&updateAll, "all", false, "update all dependencies")
	fs.BoolVar(&insecure, "precaire", false, "allow the use of insecure protocols")
}

var cmdUpdate = &cmd.Command{
	Name:      "update",
	UsageLine: "update [-all] import",
	Short:     "update a local dependency",
	Long: `gb vendor update will replaces the source with the latest available from the head of the master branch.

Updating from one copy of a dependency to another comes with several restrictions.
The first is you can only update to the head of the branch your dependency was vendered from, switching branches is not supported.
The second restriction is if you have used -tag or -revision while vendoring a dependency, your dependency is "headless"
(to borrow a term from git) and cannot be updated.

To update across branches, or from one tag/revision to another, you must first use gb vendor delete to remove the dependency, then
gb vendor fetch [-tag | -revision | -branch ] [-precaire] to replace it.

Flags:
	-all
		will update all dependencies in the manifest, otherwise only the dependency supplied.
	-precaire
		allow the use of insecure protocols.

`,
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
			err = m.RemoveDependency(d)
			if err != nil {
				return fmt.Errorf("dependency could not be deleted from manifest: %v", err)
			}

			repo, extra, err := vendor.DeduceRemoteRepo(d.Importpath, insecure)
			if err != nil {
				return fmt.Errorf("could not determine repository for import %q", d.Importpath)
			}

			wc, err := repo.Checkout(d.Branch, "", "")
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
				Importpath: d.Importpath,
				Repository: repo.URL(),
				Revision:   rev,
				Branch:     branch,
				Path:       extra,
			}

			if err := vendor.RemoveAll(filepath.Join(ctx.Projectdir(), "vendor", "src", filepath.FromSlash(d.Importpath))); err != nil {
				// TODO(dfc) need to apply vendor.cleanpath here to remove indermediate directories.
				return fmt.Errorf("dependency could not be deleted: %v", err)
			}

			dst := filepath.Join(ctx.Projectdir(), "vendor", "src", filepath.FromSlash(dep.Importpath))
			src := filepath.Join(wc.Dir(), dep.Path)

			if err := vendor.Copypath(dst, src); err != nil {
				return err
			}

			if err := m.AddDependency(dep); err != nil {
				return err
			}

			if err := vendor.WriteManifest(manifestFile(ctx), m); err != nil {
				return err
			}

			if err := wc.Destroy(); err != nil {
				return err
			}
		}

		return nil
	},
	AddFlags: addUpdateFlags,
}
