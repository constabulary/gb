package untar

import (
	"archive/tar"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// Untar extracts the contents of r to the destination dest.
// dest must not aleady exist.
func Untar(dest string, r io.Reader) error {
	parent, _ := filepath.Split(dest)
	tmpdir, err := ioutil.TempDir(parent, ".untar")
	if err != nil {
		return err
	}

	if err := untar(tmpdir, r); err != nil {
		os.RemoveAll(tmpdir)
		return err
	}

	if err := os.Rename(tmpdir, dest); err != nil {
		os.RemoveAll(tmpdir)
		return err
	}
	return nil
}

func untar(dest string, r io.Reader) error {
	tr := tar.NewReader(r)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := untarfile(dest, h, tr); err != nil {
			return err
		}
	}
	return nil
}

func untarfile(dest string, h *tar.Header, r io.Reader) error {
	path := filepath.Join(dest, h.Name)
	switch h.Typeflag {
	case tar.TypeDir:
		return os.Mkdir(path, os.FileMode(h.Mode))
	case tar.TypeReg:
		return writefile(path, r, os.FileMode(h.Mode))
	case tar.TypeXGlobalHeader:
		// ignore PAX headers
		return nil
	default:
		return errors.Errorf("unsuppored header type: %c", rune(h.Typeflag))
	}
}

func writefile(path string, r io.Reader, mode os.FileMode) error {
	dir, _ := filepath.Split(path)
	if err := os.MkdirAll(dir, mode); err != nil {
		return errors.Wrap(err, "mkdirall failed")
	}

	w, err := os.Create(path)
	if err != nil {
		return errors.Wrap(err, "could not create destination")
	}
	if _, err := io.Copy(w, r); err != nil {
		w.Close()
		return err
	}
	return w.Close()
}
