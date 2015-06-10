package main

import (
	"flag"
	"fmt"
	"os"
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

func addUpdateFlags(fs *flag.FlagSet) {
	fs.BoolVar(&updateAll, "all", false, "update all dependencies")
	fs.StringVar(&branch, "branch", "master", "branch of the package")
}

var cmdUpdate = &cmd.Command{
	Name:      "update",
	UsageLine: "update [-branch branch] [-all] import",
	Short:     "update a local dependency",
	Long: `gb vendor update will replace the source with the latest available from the head of the default upsteam branch, or the branch specified with -branch.

Flags:
	-all
		will update all depdendencies in the manifest, otherwise only the dependency supplied.

	-branch branch
		update the dependency from the named branch. If not supplied the default upstream
		branch will be used. This flag cannot be used with the -all flag.

`,
	Run: func(ctx *gb.Context, args []string) error {
		if len(args) != 1 && !updateAll {
			return fmt.Errorf("update: import path or --all flag is missing")
		} else if len(args) == 1 && updateAll {
			return fmt.Errorf("update: you cannot specify path and --all flag at once")
		} else if branch != "master" && updateAll {
			return fmt.Errorf("update: the --branch flag requires an import path to be specified (and cannot be used with the --all flag")
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
			p := d.Importpath

			err = m.RemoveDependency(d)
			if err != nil {
				return fmt.Errorf("dependency could not be deleted from manifest: %v", err)
			}

			if err := os.RemoveAll(filepath.Join(ctx.Projectdir(), "vendor", "src", filepath.FromSlash(p))); err != nil {
				// TODO(dfc) need to apply vendor.cleanpath here to remove indermediate directories.
				return fmt.Errorf("dependency could not be deleted: %v", err)
			}

			repo, extra, err := vendor.DeduceRemoteRepo(p)
			if err != nil {
				return fmt.Errorf("could not determine repository for import %q", p)
			}

			wc, err := repo.Checkout(branch, "")
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
				Importpath: p,
				Repository: url,
				Revision:   rev,
				Branch:     branch,
				Path:       extra,
			}

			if err := m.AddDependency(dep); err != nil {
				return err
			}

			dst := filepath.Join(ctx.Projectdir(), "vendor", "src", dep.Importpath)
			src := filepath.Join(wc.Dir(), dep.Path)

			if err := vendor.Copypath(dst, src); err != nil {
				return err
			}

			if err := wc.Destroy(); err != nil {
				return err
			}
		}

		return vendor.WriteManifest(manifestFile(ctx), m)
	},
	AddFlags: addUpdateFlags,
}
