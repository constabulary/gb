package vendor

import (
	"os"
	"path/filepath"
	"runtime"
)

// RemoveAll removes path and any children it contains. Unlike os.RemoveAll it
// deletes read only files on Windows.
func RemoveAll(path string) error {
	if runtime.GOOS == "windows" {
		// make sure all files are writable so we can delete them
		filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			mode := info.Mode()
			if mode|0200 == mode {
				return nil
			}
			return os.Chmod(path, mode|0200)
		})
	}
	return os.RemoveAll(path)
}
