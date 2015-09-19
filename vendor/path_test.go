// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package vendor

import (
	"os"
	"runtime"
	"testing"
)

var isReadonlyError = func(error) bool { return false }

func TestRemoveAll(t *testing.T) {
	tmpDir := os.TempDir()
	// Work directory.
	path := tmpDir + "/_TestRemoveAll_"
	fpath := path + "/file"
	dpath := path + "/dir"

	// Make directory with 1 file and remove.
	if err := os.MkdirAll(path, 0777); err != nil {
		t.Fatalf("MkdirAll %q: %s", path, err)
	}
	fd, err := os.Create(fpath)
	if err != nil {
		t.Fatalf("create %q: %s", fpath, err)
	}
	fd.Close()
	if err = RemoveAll(path); err != nil {
		t.Fatalf("RemoveAll %q (first): %s", path, err)
	}
	if _, err = os.Lstat(path); err == nil {
		t.Fatalf("Lstat %q succeeded after RemoveAll (first)", path)
	}

	// Make directory with file and subdirectory and remove.
	if err = os.MkdirAll(dpath, 0777); err != nil {
		t.Fatalf("MkdirAll %q: %s", dpath, err)
	}
	fd, err = os.Create(fpath)
	if err != nil {
		t.Fatalf("create %q: %s", fpath, err)
	}
	fd.Close()
	fd, err = os.Create(dpath + "/file")
	if err != nil {
		t.Fatalf("create %q: %s", fpath, err)
	}
	fd.Close()
	if err = RemoveAll(path); err != nil {
		t.Fatalf("RemoveAll %q (second): %s", path, err)
	}
	if _, err := os.Lstat(path); err == nil {
		t.Fatalf("Lstat %q succeeded after RemoveAll (second)", path)
	}

	// Determine if we should run the following test.
	testit := true
	if runtime.GOOS == "windows" {
		// Chmod is not supported under windows.
		testit = false
	} else {
		// Test fails as root.
		testit = os.Getuid() != 0
	}
	if testit {
		// Make directory with file and subdirectory and trigger error.
		if err = os.MkdirAll(dpath, 0777); err != nil {
			t.Fatalf("MkdirAll %q: %s", dpath, err)
		}

		for _, s := range []string{fpath, dpath + "/file1", path + "/zzz"} {
			fd, err = os.Create(s)
			if err != nil {
				t.Fatalf("create %q: %s", s, err)
			}
			fd.Close()
		}
		if err = os.Chmod(dpath, 0); err != nil {
			t.Fatalf("Chmod %q 0: %s", dpath, err)
		}

		// No error checking here: either RemoveAll
		// will or won't be able to remove dpath;
		// either way we want to see if it removes fpath
		// and path/zzz.  Reasons why RemoveAll might
		// succeed in removing dpath as well include:
		//	* running as root
		//	* running on a file system without permissions (FAT)
		RemoveAll(path)
		os.Chmod(dpath, 0777)

		for _, s := range []string{fpath, path + "/zzz"} {
			if _, err = os.Lstat(s); err == nil {
				t.Fatalf("Lstat %q succeeded after partial RemoveAll", s)
			}
		}
	}
	if err = RemoveAll(path); err != nil {
		t.Fatalf("RemoveAll %q after partial RemoveAll: %s", path, err)
	}
	if _, err = os.Lstat(path); err == nil {
		t.Fatalf("Lstat %q succeeded after RemoveAll (final)", path)
	}
}
