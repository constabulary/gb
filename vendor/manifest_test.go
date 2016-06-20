package vendor

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func mktemp(t *testing.T) string {
	s, err := mktmp()
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func assertNotExists(t *testing.T, path string) {
	_, err := os.Stat(path)
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("expected %q to be not found, got %v", path, err)
	}
}

func assertExists(t *testing.T, path string) {
	_, err := os.Stat(path)
	if err != nil {
		t.Fatalf("expected %q to be found, got %v", path, err)
	}
}

func TestManifest(t *testing.T) {
	root := mktemp(t)
	defer os.RemoveAll(root)

	mf := filepath.Join(root, "vendor")

	// check that reading an non existant manifest
	// does not return an error
	m, err := ReadManifest(mf)
	if err != nil {
		t.Fatalf("reading a non existant manifest should not fail: %v", err)
	}

	// check that no manifest file was created
	assertNotExists(t, mf)

	// add a dep
	m.Dependencies = append(m.Dependencies, Dependency{
		Importpath: "github.com/foo/bar/baz",
		Repository: "https://github.com/foo/bar",
		Revision:   "cafebad",
		Branch:     "master",
		Path:       "/baz",
	})

	// write it back
	if err := WriteManifest(mf, m); err != nil {
		t.Fatalf("WriteManifest failed: %v", err)
	}

	// check the manifest was written
	assertExists(t, mf)

	// remove it
	m.Dependencies = nil
	if err := WriteManifest(mf, m); err != nil {
		t.Fatalf("WriteManifest failed: %v", err)
	}

	// check that no manifest file was removed
	assertNotExists(t, mf)
}

func TestEmptyPathIsNotWritten(t *testing.T) {
	m := Manifest{
		Version: 0,
		Dependencies: []Dependency{{
			Importpath: "github.com/foo/bar",
			Repository: "https://github.com/foo/bar",
			Revision:   "abcdef",
			Branch:     "master",
		}},
	}
	var buf bytes.Buffer
	if err := writeManifest(&buf, &m); err != nil {
		t.Fatal(err)
	}
	want := `{
	"version": 0,
	"dependencies": [
		{
			"importpath": "github.com/foo/bar",
			"repository": "https://github.com/foo/bar",
			"revision": "abcdef",
			"branch": "master"
		}
	]
}`
	got := buf.String()
	if want != got {
		t.Fatalf("want: %s, got %s", want, got)
	}
}
