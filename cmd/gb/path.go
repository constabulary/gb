package main

import (
	"log"
	"path/filepath"

	"github.com/constabulary/gb"
)

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
			log.Fatalf("could not convert relative path %q to absolute: %v", path, err)
		}
	}
	return path
}

// isRel returns if an import path is relative or absolute.
func isRel(path string) bool {
	// TODO(dfc) should this be strings.StartsWith(".")
	return path == "."
}
