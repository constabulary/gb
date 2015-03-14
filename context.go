package gb

import "go/build"
import "path/filepath"

// Context represents an execution of one or more Targets inside a Project.
type Context struct {
	*Project
	*build.Context
}

// Srcdir returns the source directory of this context's project.
func (c *Context) Srcdir() string {
	return filepath.Join(c.Project.rootdir, "src")
}
