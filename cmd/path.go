package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/constabulary/gb"
)

// FindProjectroot works upwards from path seaching for the
// src/ directory which identifies the project root.
// If path is within GOPATH, the project root will be set to the
// matching element of GOPATH
func FindProjectroot(path string, gopaths []string) (string, error) {
	start := path
	for path != "/" {
		root := filepath.Join(path, "src")
		if _, err := os.Stat(root); err != nil {
			if os.IsNotExist(err) {
				for _, gopath := range gopaths {
					if gopath == path {
						gb.Warnf("project directory not found, falling back to $GOPATH value %q", gopath)
						return gopath, nil
					}
				}
				path = filepath.Dir(path)
				continue
			}
			return "", err
		}
		return path, nil
	}
	return "", fmt.Errorf("could not find project root in %q or its parents", start)
}
