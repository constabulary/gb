package gb

import (
	"path/filepath"
)

// Project represents a gb project. A gb project has a simlar layout to
// a $GOPATH workspace. Each gb project has a standard directory layout
// starting at the project root, which we'll refer too as $PROJECT.
//
//     $PROJECT/                       - the project root
//     $PROJECT/src/                   - base directory for the source of packages
//     $PROJECT/bin/                   - base directory for the compiled binaries
type Project struct {
	rootdir string
	srcdirs []Srcdir
}

func togopath(srcdirs []string) string {
	var s []string
	for _, srcdir := range srcdirs {
		s = append(s, filepath.Dir(srcdir))
	}
	return joinlist(s)
}

func SourceDir(root string) func(*Project) {
	return func(p *Project) {
		p.srcdirs = append(p.srcdirs, Srcdir{Root: root})
	}
}

func NewProject(root string, options ...func(*Project)) *Project {
	proj := Project{
		rootdir: root,
	}

	for _, opt := range options {
		opt(&proj)
	}

	return &proj
}

// Pkgdir returns the path to precompiled packages.
func (p *Project) Pkgdir() string {
	return filepath.Join(p.rootdir, "pkg")
}

// Projectdir returns the path root of this project.
func (p *Project) Projectdir() string {
	return p.rootdir
}

// Srcdirs returns the path to the source directories.
// The first source directory will always be
// filepath.Join(Projectdir(), "src")
// but there may be additional directories.
func (p *Project) Srcdirs() []string {
	var dirs []string
	for _, s := range p.srcdirs {
		dirs = append(dirs, s.Root)
	}
	return dirs
}

// Bindir returns the path for compiled programs.
func (p *Project) Bindir() string {
	return filepath.Join(p.rootdir, "bin")
}
