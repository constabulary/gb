package gb

import (
	"fmt"
	"go/build"
	"path/filepath"
	"sync"
	"time"
)

const debugTargetCache = false

// Context represents an execution of one or more Targets inside a Project.
type Context struct {
	*Project
	*build.Context
	workdir string

	tc Toolchain

	Statistics

	targetCache

	Force       bool // force rebuild of packages
	SkipInstall bool // do not cache compiled packages
}

// IncludePaths returns the include paths visible in this context.
func (c *Context) IncludePaths() []string {
	return []string{
		c.workdir,
		c.Pkgdir(),
	}
}

// Pkgdir returns the path to precompiled packages.
func (c *Context) Pkgdir() string {
	return filepath.Join(c.Project.Pkgdir(), c.Context.GOOS, c.Context.GOARCH)
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
		if debugTargetCache {
			Debugf("targetCache:addTargetIfMissing HIT %v", name)
		}
		return target
	}
	if debugTargetCache {
		Debugf("targetCache:addTargetIfMissing MISS %v", name)
	}
	target = f()
	c.m[name] = target
	return target
}

func (c *targetCache) targetOrMissing(name string, f func() Target) Target {
	c.Lock()
	target, ok := c.m[name]
	c.Unlock()
	if ok {
		if debugTargetCache {
			Debugf("targetCache:targetOrMissing HIT %v", name)
		}
		return target
	}
	if debugTargetCache {
		Debugf("targetCache:targetOrMissing MISS %v", name)
	}
	return f()
}

// Statistics records the various Durations
type Statistics struct {
	sync.Mutex
	stats map[string]time.Duration
}

func (s *Statistics) Record(name string, d time.Duration) {
	s.Lock()
	defer s.Unlock()
	if s.stats == nil {
		s.stats = make(map[string]time.Duration)
	}
	s.stats[name] += d
}

func (s *Statistics) Total() time.Duration {
	s.Lock()
	defer s.Unlock()
	var d time.Duration
	for _, v := range s.stats {
		d += v
	}
	return d
}

func (s *Statistics) String() string {
	s.Lock()
	defer s.Unlock()
	return fmt.Sprintf("%v", s.stats)
}
