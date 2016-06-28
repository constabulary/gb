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
type Project interface {

	// Projectdir returns the path root of this project.
	Projectdir() string

	// Pkgdir returns the path to precompiled packages.
	Pkgdir() string

	// Bindir returns the path for compiled programs.
	Bindir() string
}

type project struct {
	rootdir string
}

func NewProject(root string) Project {
	proj := project{
		rootdir: root,
	}
	return &proj
}

// Pkgdir returns the path to precompiled packages.
func (p *project) Pkgdir() string {
	return filepath.Join(p.rootdir, "pkg")
}

// Projectdir returns the path root of this project.
func (p *project) Projectdir() string {
	return p.rootdir
}

// Bindir returns the path for compiled programs.
func (p *project) Bindir() string {
	return filepath.Join(p.rootdir, "bin")
}
