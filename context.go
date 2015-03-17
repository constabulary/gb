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

	tc Toolchain

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
	m map[string]Target
}

func (c *targetCache) addTargetIfMissing(name string, f func() Target) Target {
	c.Lock()
	defer c.Unlock()
	if c.m == nil {
		c.m = make(map[string]Target)
	}
	target, ok := c.m[name]
	if ok {
		Debugf("targetCache:addTargetIdMissing HIT %v", name)
		return target
	}
	Debugf("targetCache:addTargetIfMissing MISS %v", name)
	target = f()
	c.m[name] = target
	return target
}

func (c *targetCache) targetOrMissing(name string, f func() Target) Target {
	c.Lock()
	target, ok := c.m[name]
	c.Unlock()
	if ok {
		Debugf("targetCache:targetOrMissing HIT %v", name)
		return target
	}
	Debugf("targetCache:targetOrMissing MISS %v", name)
	return f()
}
