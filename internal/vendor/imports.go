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

	"github.com/pkg/errors"
)

// ParseImports parses Go packages from a specific root returning a set of import paths.
func ParseImports(root string) (map[string]bool, error) {
	pkgs := make(map[string]bool)

	var walkFn = func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") || name == "testdata" {
				return filepath.SkipDir
			}
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
			pkgs[p] = true
		}
		return nil
	}

	err := filepath.Walk(root, walkFn)
	return pkgs, err
}

// FetchMetadata fetchs the remote metadata for path.
func FetchMetadata(path string, insecure bool) (rc io.ReadCloser, err error) {
	defer func() {
		if err != nil {
			err = errors.Wrap(err, "unable to determine remote metadata protocol")
		}
	}()
	// try https first
	rc, err = fetchMetadata("https", path)
	if err == nil {
		return
	}
	// try http if supported
	if insecure {
		rc, err = fetchMetadata("http", path)
	}
	return
}

func fetchMetadata(scheme, path string) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s://%s?go-get=1", scheme, path)
	switch scheme {
	case "https", "http":
		resp, err := http.Get(url)
		if err != nil {
			return nil, fmt.Errorf("failed to access url %q", url)
		}
		return resp.Body, nil
	default:
		return nil, fmt.Errorf("unknown remote protocol scheme: %q", scheme)
	}
}

// ParseMetadata fetchs and decodes remote metadata for path.
func ParseMetadata(path string, insecure bool) (string, string, string, error) {
	rc, err := FetchMetadata(path, insecure)
	if err != nil {
		return "", "", "", err
	}
	defer rc.Close()

	imports, err := parseMetaGoImports(rc)
	if err != nil {
		return "", "", "", err
	}
	match := -1
	for i, im := range imports {
		if !strings.HasPrefix(path, im.Prefix) {
			continue
		}
		if match != -1 {
			return "", "", "", fmt.Errorf("multiple meta tags match import path %q", path)
		}
		match = i
	}
	if match == -1 {
		return "", "", "", fmt.Errorf("go-import metadata not found")
	}
	return imports[match].Prefix, imports[match].VCS, imports[match].RepoRoot, nil
}
