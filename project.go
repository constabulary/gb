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

	// Srcdirs returns the path to the source directories.
	Srcdirs() []string
}

type project struct {
	rootdir string
	srcdirs []string
}

func NewProject(root string) Project {
	proj := project{
		rootdir: root,
		srcdirs: []string{
			filepath.Join(root, "src"),
			filepath.Join(root, "vendor", "src"),
		},
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

// Srcdirs returns the path to the source directories.
// The first source directory will always be
// filepath.Join(Projectdir(), "src")
// but there may be additional directories.
func (p *project) Srcdirs() []string {
	return p.srcdirs
}

// Bindir returns the path for compiled programs.
func (p *project) Bindir() string {
	return filepath.Join(p.rootdir, "bin")
}
