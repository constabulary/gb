package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
	"github.com/constabulary/gb/cmd/gb-vendor/vendor"
)

var cmdPurge = &cmd.Command{
	Name:      "purge",
	UsageLine: "purge",
	Short:     "purges all unreferenced dependencies",
	Long: `gb vendor purge will remove all unreferenced dependencies

`,
	Run: func(ctx *gb.Context, args []string) error {
		m, err := vendor.ReadManifest(manifestFile(ctx))
		if err != nil {
			return fmt.Errorf("could not load manifest: %v", err)
		}

		imports, err := vendor.ParseImports(ctx.Projectdir())
		if err != nil {
			return fmt.Errorf("import could not be parsed: %v", err)
		}

		var hasImportWithPrefix = func(d string) bool {
			for i := range imports {
				if strings.HasPrefix(i, d) {
					return true
				}
			}
			return false
		}

		dependencies := make([]vendor.Dependency, len(m.Dependencies))
		copy(dependencies, m.Dependencies)

		for _, d := range dependencies {
			if !hasImportWithPrefix(d.Importpath) {
				dep, err := m.GetDependencyForImportpath(d.Importpath)
				if err != nil {
					return fmt.Errorf("could not get get dependency: %v", err)
				}

				if err := m.RemoveDependency(dep); err != nil {
					return fmt.Errorf("dependency could not be removed: %v", err)
				}
				if err := os.RemoveAll(filepath.Join(ctx.Projectdir(), "vendor", "src", filepath.FromSlash(d.Importpath))); err != nil {
					// TODO(dfc) need to apply vendor.cleanpath here to remove indermediate directories.
					return fmt.Errorf("dependency could not be deleted: %v", err)
				}
			}
		}

		return vendor.WriteManifest(manifestFile(ctx), m)
	},
}
