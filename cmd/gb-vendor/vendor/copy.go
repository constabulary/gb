package vendor

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const debugCopypath = true
const debugCopyfile = false

// Copypath copies the contents of src to dst, excluding any file or
// directory that starts with a period.
func Copypath(dst string, src string) error {
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasPrefix(filepath.Base(path), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}

		if info.Mode()&os.ModeSymlink != 0 {
			if debugCopypath {
				fmt.Printf("skiping symlink: %v\n", path)
			}
			return nil
		}

		dst := filepath.Join(dst, path[len(src):])
		return copyfile(dst, path)
	})
	if err != nil {
		// if there was an error during copying, remove the partial copy.
		os.RemoveAll(dst)
	}
	return err
}

func copyfile(dst, src string) error {
	err := mkdir(filepath.Dir(dst))
	if err != nil {
		return fmt.Errorf("copyfile: mkdirall: %v", err)
	}
	r, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("copyfile: open(%q): %v", src, err)
	}
	defer r.Close()
	w, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("copyfile: create(%q): %v", dst, err)
	}
	if debugCopyfile {
		fmt.Printf("copyfile(dst: %v, src: %v)\n", dst, src)
	}
	_, err = io.Copy(w, r)
	return err
}
