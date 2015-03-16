package gb

import (
	"go/build"
	"path/filepath"
	"sync"
)

// Context represents an execution of one or more Targets inside a Project.
type Context struct {
	*Project
	*build.Context

	Statistics

	targetCache
}

// Srcdir returns the source directory of this context's project.
func (c *Context) Srcdir() string {
	return filepath.Join(c.Project.rootdir, "src")
}

// ResolvePackage resolves the package at path using the current context.
func (c *Context) ResolvePackage(path string) *Package {
	return resolvePackage(c, path)
}

// Destroy removes the temporary working files of this context.
func (c *Context) Destroy() error { return nil }

type targetCache struct {
	sync.Mutex
	m map[*Package]Target
}

func (c *targetCache) addTargetIfMissing(pkg *Package, f func() Target) Target {
	c.Lock()
	defer c.Unlock()
	if c.m == nil {
		c.m = make(map[*Package]Target)
	}
	target, ok := c.m[pkg]
	if !ok {
		target = f()
		c.m[pkg] = target
	}
	return target
}
