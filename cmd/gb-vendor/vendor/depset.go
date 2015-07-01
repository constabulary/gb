package vendor

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"strings"
)

// Pkg describes a Go package.
type Pkg struct {
	*Depset
	*build.Package
}

// Depset describes a set of related Go packages.
type Depset struct {
	Root   string
	Prefix string
	Pkgs   map[string]*Pkg
}

// LoadPaths returns a map of paths to Depsets.
func LoadPaths(paths ...struct{ Root, Prefix string }) (map[string]*Depset, error) {
	m := make(map[string]*Depset)
	for _, p := range paths {
		set, err := LoadTree(p.Root, p.Prefix)
		if err != nil {
			return nil, err
		}
		m[set.Root] = set
	}
	return m, nil
}

// LoadTree parses a tree of source files into a map of *pkgs.
func LoadTree(root string, prefix string) (*Depset, error) {
	d := Depset{
		Root:   root,
		Prefix: prefix,
		Pkgs:   make(map[string]*Pkg),
	}
	fn := func(dir string, fi os.FileInfo) error {
		importpath := filepath.Join(prefix, dir[len(root)+1:])

		// if we're at the root of a tree, skip it
		if importpath == "" {
			return nil
		}

		p, err := loadPackage(&d, dir)
		if err != nil {
			if _, ok := err.(*build.NoGoError); ok {
				return nil
			}
			return fmt.Errorf("loadPackage(%q, %q): %v", dir, importpath, err)
		}
		p.ImportPath = filepath.ToSlash(importpath)
		if p != nil {
			d.Pkgs[p.ImportPath] = p
		}
		return nil
	}

	// handle root of the tree
	fi, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if err := fn(root+string(filepath.Separator), fi); err != nil {
		return nil, err
	}

	// walk sub directories
	err = eachDir(root, fn)
	return &d, err
}

func loadPackage(d *Depset, dir string) (*Pkg, error) {
	p := Pkg{
		Depset: d,
	}
	var err error

	// expolit local import logic
	p.Package, err = build.ImportDir(dir, build.ImportComment)
	return &p, err
}

func eachDir(dir string, fn func(string, os.FileInfo) error) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	files, err := f.Readdir(-1)
	for _, fi := range files {
		if !fi.IsDir() {
			continue
		}
		if strings.HasPrefix(fi.Name(), "_") || strings.HasPrefix(fi.Name(), ".") || fi.Name() == "testdata" {
			continue
		}
		path := filepath.Join(dir, fi.Name())
		if err := fn(path, fi); err != nil {
			return err
		}
		if err := eachDir(path, fn); err != nil {
			return err
		}
	}
	return nil
}
