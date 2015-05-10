package gb

import (
	"path/filepath"
)

// Project represents a gb project. A gb project has a simlar layout to
// a $GOPATH workspace. Each gb project has a standard directory layout
// starting at the project root, which we'll refer too as $PROJECT.
//
//     $PROJECT/                       - the project root
//     $PROJECT/.gogo/                 - used internally by gogo and identifies
//                                       the root of the project.
//     $PROJECT/src/                   - base directory for the source of packages
//     $PROJECT/bin/                   - base directory for the compiled binaries
type Project struct {
	rootdir string
}

func togopath(srcdirs []string) string {
	var s []string
	for _, srcdir := range srcdirs {
		s = append(s, filepath.Dir(srcdir))
	}
	return joinlist(s)
}

func NewProject(root string) *Project {
	return &Project{
		rootdir: root,
	}
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
	return append(p.srcdir(), filepath.Join(p.Projectdir(), "vendor", "src"))
}

// srcdir returns the source directory of the project.
func (p *Project) srcdir() []string {
	return []string{filepath.Join(p.Projectdir(), "src")}
}

// Bindir returns the path for compiled programs.
func (p *Project) Bindir() string {
	return filepath.Join(p.rootdir, "bin")
}
