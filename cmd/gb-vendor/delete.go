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
	// gb vendor delete flags

	// delete all dependencies
	deleteAll bool
)

func addDeleteFlags(fs *flag.FlagSet) {
	fs.BoolVar(&deleteAll, "all", false, "delete all dependencies")
}

var cmdDelete = &cmd.Command{
	Name:      "delete",
	UsageLine: "delete [-all] importpath",
	Short:     "deletes a local dependency",
	Long: `delete removes a dependency from $PROJECT/vendor/src and the vendor manifest

Flags:
	-all
		remove all dependencies

`,
	Run: func(ctx *gb.Context, args []string) error {
		if len(args) != 1 && !deleteAll {
			return fmt.Errorf("delete: import path or --all flag is missing")
		} else if len(args) == 1 && deleteAll {
			return fmt.Errorf("delete: you cannot specify path and --all flag at once")
		}

		m, err := vendor.ReadManifest(manifestFile(ctx))
		if err != nil {
			return fmt.Errorf("could not load manifest: %v", err)
		}

		var dependencies []vendor.Dependency
		if deleteAll {
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
			path := d.Importpath

			if err := m.RemoveDependency(d); err != nil {
				return fmt.Errorf("dependency could not be deleted: %v", err)
			}

			if err := os.RemoveAll(filepath.Join(ctx.Projectdir(), "vendor", "src", filepath.FromSlash(path))); err != nil {
				// TODO(dfc) need to apply vendor.cleanpath here to remove indermediate directories.
				return fmt.Errorf("dependency could not be deleted: %v", err)
			}
		}
		return vendor.WriteManifest(manifestFile(ctx), m)
	},
	AddFlags: addDeleteFlags,
}
