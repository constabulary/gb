package main

import (
	"path/filepath"
	"strings"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
	"github.com/constabulary/gb/internal/fileutils"
	"github.com/constabulary/gb/internal/vendor"
	"github.com/pkg/errors"
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
			return errors.Wrap(err, "could not load manifest")
		}

		imports, err := vendor.ParseImports(ctx.Projectdir())
		if err != nil {
			return errors.Wrap(err, "import could not be parsed")
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
					return errors.Wrap(err, "could not get get dependency")
				}

				if err := m.RemoveDependency(dep); err != nil {
					return errors.Wrap(err, "dependency could not be removed")
				}
				if err := fileutils.RemoveAll(filepath.Join(ctx.Projectdir(), "vendor", "src", filepath.FromSlash(d.Importpath))); err != nil {
					// TODO(dfc) need to apply vendor.cleanpath here to remove intermediate directories.
					return errors.Wrap(err, "dependency could not be deleted")
				}
			}
		}

		return vendor.WriteManifest(manifestFile(ctx), m)
	},
}
