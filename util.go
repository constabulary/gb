package gb

import "io"
import "os"
import "path/filepath"

func copyfile(dst, src string) error {
	err := os.MkdirAll(filepath.Dir(dst), 0755)
	if err != nil {
		return err
	}
	r, err := os.Open(src)
	if err != nil {
		return err
	}
	defer r.Close()
	w, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, r)
	return err
}
