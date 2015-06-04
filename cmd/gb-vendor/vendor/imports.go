package vendor

import (
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"net/http"
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

// FetchMetadata fetchs the remote metadata for path.
func FetchMetadata(path string) (io.ReadCloser, error) {
	url := fmt.Sprintf("https://%s?go-get=1", path)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// ParseMetadata fetchs and decodes remote metadata for path.
func ParseMetadata(path string) (string, string, string, error) {
	rc, err := FetchMetadata(path)
	if err != nil {
		return "", "", "", err
	}
	defer rc.Close()

	meta, err := parseMetaGoImports(rc)
	if len(meta) < 1 {
		return "", "", "", fmt.Errorf("go-import metadata not found")
	}
	if err != nil {
		return "", "", "", err
	}
	return meta[0].Prefix, meta[0].VCS, meta[0].RepoRoot, nil
}
