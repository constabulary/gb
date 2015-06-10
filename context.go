package gb

import (
	"bytes"
	"fmt"
	"go/build"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

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

	permits chan bool // used to limit concurrency of Run targets

	ldflags []string // flags passed to the linker
}

// NewContext returns a new build context from this project.
// By default this context will use the gc toolchain with the
// host's GOOS and GOARCH values.
func (p *Project) NewContext(opts ...func(*Context) error) (*Context, error) {
	bc := build.Default
	bc.GOPATH = togopath(p.Srcdirs())
	defaults := []func(*Context) error{
		GcToolchain(),
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

// Ldflags sets options passed to the linker.
func Ldflags(flags string) func(*Context) error {
	return func(c *Context) error {
		var err error
		c.ldflags, err = splitQuotedFields(flags)
		return err
	}
}

func newContext(p *Project, bc *build.Context) *Context {
	permits := make(chan bool, runtime.NumCPU())
	for i := cap(permits); i > 0; i-- {
		permits <- true
	}
	return &Context{
		Project: p,
		Context: bc,
		workdir: mktmpdir(),
		pkgs:    make(map[string]*Package),
		permits: permits,
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

// Workdir returns the path to this Context's working directory.
func (c *Context) Workdir() string { return c.workdir }

// ResolvePackage resolves the package at path using the current context.
func (c *Context) ResolvePackage(path string) (*Package, error) {
	return c.loadPackage(nil, path)
}

// ResolvePackageWithTests resolves the package at path using the current context
// it also resolves the internal and external test dependenices, although these are
// not returned, only cached in the Context.
func (c *Context) ResolvePackageWithTests(path string) (*Package, error) {
	p, err := c.ResolvePackage(path)
	if err != nil {
		return nil, err
	}
	var imports []string
	imports = append(imports, p.Package.TestImports...)
	imports = append(imports, p.Package.XTestImports...)
	for _, i := range imports {
		_, err := c.ResolvePackage(i)
		if err != nil {
			return nil, err
		}
	}
	return p, nil
}

// loadPackage recursively resolves path and its imports and if successful
// stores those packages in the Context's internal package cache.
func (c *Context) loadPackage(stack []string, path string) (*Package, error) {
	if build.IsLocalImport(path) {
		// sanity check
		return nil, fmt.Errorf("%q is not a valid import path", path)
	}
	if pkg, ok := c.pkgs[path]; ok {
		// already loaded, just return
		return pkg, nil
	}

	push := func(path string) {
		stack = append(stack, path)
	}
	pop := func(path string) {
		stack = stack[:len(stack)-1]
	}
	onStack := func(path string) bool {
		for _, p := range stack {
			if p == path {
				return true
			}
		}
		return false
	}

	p, err := c.Context.Import(path, c.Projectdir(), 0)
	if err != nil {
		return nil, err
	}
	push(path)
	var stale bool
	for _, i := range p.Imports {
		if Stdlib[i] {
			continue
		}
		if onStack(i) {
			push(i)
			return nil, fmt.Errorf("import cycle detected: %s", strings.Join(stack, " -> "))
		}
		pkg, err := c.loadPackage(stack, i)
		if err != nil {
			return nil, err
		}
		stale = stale || pkg.Stale
	}
	pop(path)

	pkg := Package{
		Context: c,
		Package: p,
	}
	pkg.Stale = stale || isStale(&pkg)
	Debugf("loadPackage: %v %v (%v)", path, pkg.Stale, pkg.Dir)
	c.pkgs[path] = &pkg
	return &pkg, nil
}

// Destroy removes the temporary working files of this context.
func (c *Context) Destroy() error {
	Debugf("removing work directory: %v", c.workdir)
	return os.RemoveAll(c.workdir)
}

// Run returns a Target representing the result of executing a CmdTarget.
func (c *Context) Run(cmd *exec.Cmd, deps ...Target) Target {
	annotate := func() error {
		<-c.permits
		Debugf("run %v", cmd.Args)
		err := cmd.Run()
		c.permits <- true
		if err != nil {
			err = fmt.Errorf("run %v: %v", cmd.Args, err)
		}
		return err
	}
	target := newTarget(annotate, deps...)
	return &target // TODO
}

func (c *Context) run(dir string, env []string, command string, args ...string) error {
	var buf bytes.Buffer
	err := c.runOut(&buf, dir, env, command, args...)
	if err != nil {
		return fmt.Errorf("# %s %s: %v\n%s", command, strings.Join(args, " "), err, buf.String())
	}
	return nil
}

func (c *Context) runOut(output io.Writer, dir string, env []string, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	cmd.Stdout = output
	cmd.Stderr = os.Stderr
	cmd.Env = mergeEnvLists(env, envForDir(cmd.Dir))
	<-c.permits
	Debugf("cd %s; %s", cmd.Dir, cmd.Args)
	c.permits <- true
	err := cmd.Run()
	return err
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

// envForDir returns a copy of the environment
// suitable for running in the given directory.
// The environment is the current process's environment
// but with an updated $PWD, so that an os.Getwd in the
// child will be faster.
func envForDir(dir string) []string {
	env := os.Environ()
	// Internally we only use rooted paths, so dir is rooted.
	// Even if dir is not rooted, no harm done.
	return mergeEnvLists([]string{"PWD=" + dir}, env)
}

// mergeEnvLists merges the two environment lists such that
// variables with the same name in "in" replace those in "out".
func mergeEnvLists(in, out []string) []string {
NextVar:
	for _, inkv := range in {
		k := strings.SplitAfterN(inkv, "=", 2)[0]
		for i, outkv := range out {
			if strings.HasPrefix(outkv, k) {
				out[i] = inkv
				continue NextVar
			}
		}
		out = append(out, inkv)
	}
	return out
}
