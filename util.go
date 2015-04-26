package gb

import "io"
import "os"
import "path/filepath"
import "fmt"

const debugCopyfile = false

func copyfile(dst, src string) error {
	err := os.MkdirAll(filepath.Dir(dst), 0755)
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
		Debugf("copyfile(dst: %v, src: %v)", dst, src)
	}
	_, err = io.Copy(w, r)
	return err
}
