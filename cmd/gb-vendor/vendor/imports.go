package vendor

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/constabulary/gb"
)

// ParseImports parses Go packages from a specific root returning a set of import paths.
func ParseImports(root string) (map[string]bool, error) {
	pkgs := make(map[string]bool)

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
			p := strings.Replace(s.Path.Value, "\"", "", -1)
			if !contains(gb.Stdlib, p) {
				pkgs[p] = true
			}
		}
		return nil
	}

	err := filepath.Walk(root, walkFn)
	return pkgs, err
}
