package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// FindProjectroot works upwards from path seaching for the
// src/ directory which identifies the project root.
func FindProjectroot(path string) (string, error) {
	if path == "" {
		return "", errors.New("project root is blank")
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
		return path, nil
	}
	return "", fmt.Errorf(`could not find project root in "%s" or its parents`, start)
}
