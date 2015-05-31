package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/constabulary/gb"
)

// FindProjectroot works upwards from path seaching for the
// src/ directory which identifies the project root.
func FindProjectroot(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("project root is blank")
	}
	start := path
	for path != "/" {
		root := filepath.Join(path, "src")
		if _, err := os.Stat(root); err != nil {
			if os.IsNotExist(err) {
				path = filepath.Dir(path)
				continue
			}
			return "", err
		}
		path, err := filepath.EvalSymlinks(path)
		if err != nil {
			return "", err
		}
		return path, nil
	}
	return "", fmt.Errorf("could not find project root in %q or its parents", start)
}

// RelImportPaths converts a list of potentially relative import path (a path starting with .)
// to an absolute import path relative to the project root of the Context provided.
func RelImportPaths(ctx *gb.Context, paths ...string) []string {
	for i := 0; i < len(paths); i++ {
		paths[i] = relImportPath(ctx.Srcdirs()[0], paths[i])
	}
	return paths
}

func relImportPath(root, path string) string {
	if isRel(path) {
		var err error
		path, err = filepath.Rel(root, path)
		if err != nil {
			gb.Fatalf("could not convert relative path %q to absolute: %v", path, err)
		}
	}
	return path
}

// isRel returns if an import path is relative or absolute.
func isRel(path string) bool {
	// TODO(dfc) should this be strings.StartsWith(".")
	return path == "."
}
