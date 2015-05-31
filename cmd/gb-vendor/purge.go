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

func parseImports(root string) (map[string]struct{}, error) {
	var found = struct{}{}            // Does not take any space
	pkgs := make(map[string]struct{}) // Set

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
			if _, ok := gb.Stdlib[s.Path.Value]; !ok {
				pkgs[strings.Replace(s.Path.Value, "\"", "", -1)] = found
			}
		}
		return nil
	}

	if err := filepath.Walk(root, walkFn); err != nil {
		return pkgs, err
	}

	return pkgs, nil
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

		dependencies := make([]vendor.Dependency, len(m.Dependencies))
		copy(dependencies, m.Dependencies)

		for _, d := range dependencies {
			if _, ok := imports[d.Importpath]; !ok {
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
