package cmd

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindProjectroot works upwards from path seaching for the
// src/ directory which identifies the project root.
func FindProjectroot(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("project root is blank")
	}
	start := path
	for path != filepath.Dir(path) {
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
