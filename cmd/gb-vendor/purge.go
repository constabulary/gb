package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
	"github.com/constabulary/gb/cmd/gb-vendor/vendor"
)

func init() {
	registerCommand(PurgeCmd)
}

func parseImports(root string) (map[string]bool, error) {
	pkgs := make(map[string]bool) // Set

	var walkFn = func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" { // Parse only go source files
			return nil
		}

		fs := token.NewFileSet()
		f, err := parser.ParseFile(fs, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}

		for _, s := range f.Imports {
			if !gb.Stdlib[s.Path.Value] {
				pkgs[strings.Replace(s.Path.Value, "\"", "", -1)] = true
			}
		}
		return nil
	}

	err := filepath.Walk(root, walkFn)
	return pkgs, err
}

var PurgeCmd = &cmd.Command{
	Name:      "purge",
	ShortDesc: "purges all unreferenced dependencies",
	Run: func(ctx *gb.Context, args []string) error {
		m, err := vendor.ReadManifest(manifestFile(ctx))
		if err != nil {
			return fmt.Errorf("could not load manifest: %v", err)
		}

		imports, err := parseImports(ctx.Projectdir())
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

				localClone := vendor.GitClone{
					Path: filepath.Join(ctx.Projectdir(), "vendor", "src", dep.Importpath),
				}
				if err := localClone.Destroy(); err != nil {
					return fmt.Errorf("dependency could not be deleted: %v", err)
				}
			}
		}

		return vendor.WriteManifest(manifestFile(ctx), m)
	},
}
