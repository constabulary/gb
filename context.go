package gb

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

	Force       bool // force rebuild of packages
	SkipInstall bool // do not cache compiled packages

	pkgs map[string]*Package // map of package paths to resolved packages
}

// NullToolchain configures the Context to use the null toolchain.
func NullToolchain(c *Context) error {
	c.tc = new(nulltoolchain)
	return nil
}

// NewContext returns a new build context from this project.
// By default this context will use the gc toolchain with the
// host's GOOS and GOARCH values.
func (p *Project) NewContext(opts ...func(*Context) error) (*Context, error) {
	bc := build.Default
	bc.GOPATH = togopath(p.Srcdirs())
	defaults := []func(*Context) error{
		GcToolchain(bc.GOROOT),
	}
	ctx := newContext(p, &bc)
	for _, opt := range append(defaults, opts...) {
		err := opt(ctx)
		if err != nil {
			return nil, err
		}
	}
	return ctx, nil
}

func newContext(p *Project, bc *build.Context) *Context {
	return &Context{
		Project: p,
		Context: bc,
		workdir: mktmpdir(),
		pkgs:    make(map[string]*Package),
	}
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
	// TODO(dfc) c.Context.{GOOS,GOARCH} may be out of date wrt. tc.{goos,goarch}
	return filepath.Join(c.Project.Pkgdir(), c.Context.GOOS, c.Context.GOARCH)
}

// ResolvePackage resolves the package at path using the current context.
func (c *Context) ResolvePackage(path string) (*Package, error) {
	return c.loadPackage(make(map[string]bool), path)
}

// loadPackage recursively resolves path and its imports and if successful
// stores those packages in the Context's internal package cache.
func (c *Context) loadPackage(stack map[string]bool, path string) (*Package, error) {
	if build.IsLocalImport(path) {
		// sanity check
		return nil, fmt.Errorf("%q is not a valid import path", path)
	}
	if pkg, ok := c.pkgs[path]; ok {
		// already loaded, just return
		return pkg, nil
	}
	Debugf("loadPackage: %v", path)

	push := func(path string) {
		stack[path] = true
	}
	pop := func(path string) {
		delete(stack, path)
	}

	p, err := c.Context.Import(path, c.Projectdir(), 0)
	if err != nil {
		return nil, err
	}
	push(path)
	var stale bool
	for _, i := range p.Imports {
		if stdlib[i] {
			continue
		}
		pkg, err := c.loadPackage(stack, i)
		if err != nil {
			return nil, err
		}
		stale = stale || pkg.Stale
	}
	pop(path)

	pkg := Package{
		ctx:     c,
		Package: p,
	}
	pkg.Stale = stale || isStale(&pkg)
	c.pkgs[path] = &pkg
	return &pkg, nil
}

// Destroy removes the temporary working files of this context.
func (c *Context) Destroy() error {
	Debugf("removing work directory: %v", c.workdir)
	return os.RemoveAll(c.workdir)
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

// matchPattern(pattern)(name) reports whether
// name matches pattern.  Pattern is a limited glob
// pattern in which '...' means 'any string' and there
// is no other special syntax.
func matchPattern(pattern string) func(name string) bool {
	re := regexp.QuoteMeta(pattern)
	re = strings.Replace(re, `\.\.\.`, `.*`, -1)
	// Special case: foo/... matches foo too.
	if strings.HasSuffix(re, `/.*`) {
		re = re[:len(re)-len(`/.*`)] + `(/.*)?`
	}
	reg := regexp.MustCompile(`^` + re + `$`)
	return func(name string) bool {
		return reg.MatchString(name)
	}
}

// hasPathPrefix reports whether the path s begins with the
// elements in prefix.
func hasPathPrefix(s, prefix string) bool {
	switch {
	default:
		return false
	case len(s) == len(prefix):
		return s == prefix
	case len(s) > len(prefix):
		if prefix != "" && prefix[len(prefix)-1] == '/' {
			return strings.HasPrefix(s, prefix)
		}
		return s[len(prefix)] == '/' && s[:len(prefix)] == prefix
	}
}

// treeCanMatchPattern(pattern)(name) reports whether
// name or children of name can possibly match pattern.
// Pattern is the same limited glob accepted by matchPattern.
func treeCanMatchPattern(pattern string) func(name string) bool {
	wildCard := false
	if i := strings.Index(pattern, "..."); i >= 0 {
		wildCard = true
		pattern = pattern[:i]
	}
	return func(name string) bool {
		return len(name) <= len(pattern) && hasPathPrefix(pattern, name) ||
			wildCard && strings.HasPrefix(name, pattern)
	}
}

// AllPackages returns all the packages that can be found under the $PROJECT/src directory.
// The pattern is a path including "...".
func (c *Context) AllPackages(pattern string) []string {
	return matchPackages(c, pattern)
}

func matchPackages(c *Context, pattern string) []string {
	Debugf("matchPackages: %v %v", c.srcdir(), pattern)
	match := func(string) bool { return true }
	treeCanMatch := func(string) bool { return true }
	if pattern != "all" && pattern != "std" {
		match = matchPattern(pattern)
		treeCanMatch = treeCanMatchPattern(pattern)
	}

	var pkgs []string

	for _, src := range c.srcdir() {
		src = filepath.Clean(src) + string(filepath.Separator)
		filepath.Walk(src, func(path string, fi os.FileInfo, err error) error {
			if err != nil || !fi.IsDir() || path == src {
				return nil
			}

			// Avoid .foo, _foo, and testdata directory trees.
			_, elem := filepath.Split(path)
			if strings.HasPrefix(elem, ".") || strings.HasPrefix(elem, "_") || elem == "testdata" {
				return filepath.SkipDir
			}

			name := filepath.ToSlash(path[len(src):])
			if pattern == "std" && strings.Contains(name, ".") {
				return filepath.SkipDir
			}
			if !treeCanMatch(name) {
				return filepath.SkipDir
			}
			if !match(name) {
				return nil
			}
			_, err = c.Context.Import(".", path, 0)
			if err != nil {
				if _, noGo := err.(*build.NoGoError); noGo {
					return nil
				}
			}
			pkgs = append(pkgs, name)
			return nil
		})
	}
	return pkgs
}
