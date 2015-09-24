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
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/constabulary/gb/log"
)

// Context represents an execution of one or more Targets inside a Project.
type Context struct {
	*Project
	Context *build.Context
	workdir string

	tc Toolchain

	gohostos, gohostarch     string // GOOS and GOARCH for this host
	gotargetos, gotargetarch string // GOOS and GOARCH for the target

	Statistics

	Force       bool // force rebuild of packages
	SkipInstall bool // do not cache compiled packages

	pkgs map[string]*Package // map of package paths to resolved packages

	gcflags []string // flags passed to the compiler
	ldflags []string // flags passed to the linker

	linkmode, buildmode string // link and build modes

	buildtags []string // build tags
}

// GOOS configures the Context to use goos as the target os.
func GOOS(goos string) func(*Context) error {
	return func(c *Context) error {
		if goos == "" {
			return fmt.Errorf("goos cannot be blank")
		}
		c.gotargetos = goos
		return nil
	}
}

// GOARCH configures the Context to use goarch as the target arch.
func GOARCH(goarch string) func(*Context) error {
	return func(c *Context) error {
		if goarch == "" {
			return fmt.Errorf("goarch cannot be blank")
		}
		c.gotargetarch = goarch
		return nil
	}
}

// Tags configured the context to use these additional build tags
func Tags(tags ...string) func(*Context) error {
	return func(c *Context) error {
		c.buildtags = make([]string, len(tags))
		copy(c.buildtags, tags)
		return nil
	}
}

// NewContext returns a new build context from this project.
// By default this context will use the gc toolchain with the
// host's GOOS and GOARCH values.
func (p *Project) NewContext(opts ...func(*Context) error) (*Context, error) {
	if len(p.srcdirs) == 0 {
		return nil, fmt.Errorf("no source directories supplied")
	}
	envOr := func(key, def string) string {
		if v := os.Getenv(key); v != "" {
			return v
		} else {
			return def
		}
	}

	defaults := []func(*Context) error{
		// must come before GcToolchain()
		func(c *Context) error {
			c.gohostos = runtime.GOOS
			c.gohostarch = runtime.GOARCH
			c.gotargetos = envOr("GOOS", runtime.GOOS)
			c.gotargetarch = envOr("GOARCH", runtime.GOARCH)
			return nil
		},
		GcToolchain(),
	}
	ctx := Context{
		Project:   p,
		workdir:   mktmpdir(),
		pkgs:      make(map[string]*Package),
		buildmode: "exe",
	}

	for _, opt := range append(defaults, opts...) {
		err := opt(&ctx)
		if err != nil {
			return nil, err
		}
	}

	// sort build tags to ensure the ctxSring and Suffix is stable
	sort.Strings(ctx.buildtags)

	// backfill enbedded go/build.Context
	ctx.Context = &build.Context{
		GOOS:     ctx.gotargetos,
		GOARCH:   ctx.gotargetarch,
		GOROOT:   runtime.GOROOT(),
		GOPATH:   togopath(p.Srcdirs()),
		Compiler: runtime.Compiler, // TODO(dfc) probably unused

		// Make sure we use the same set of release tags as go/build
		ReleaseTags: build.Default.ReleaseTags,
		BuildTags:   ctx.buildtags,

		CgoEnabled: build.Default.CgoEnabled,
	}
	return &ctx, nil
}

// Gcflags sets options passed to the compiler.
func Gcflags(flags ...string) func(*Context) error {
	return func(c *Context) error {
		c.gcflags = flags
		return nil
	}
}

// Ldflags sets options passed to the linker.
func Ldflags(flags ...string) func(*Context) error {
	return func(c *Context) error {
		c.ldflags = flags
		return nil
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
	return filepath.Join(c.Project.Pkgdir(), c.ctxString())
}

// Suffix returns the suffix (if any) for binaries produced
// by this context.
func (c *Context) Suffix() string {
	suffix := c.ctxString()
	if suffix != "" {
		suffix = "-" + suffix
	}
	return suffix
}

// Workdir returns the path to this Context's working directory.
func (c *Context) Workdir() string { return c.workdir }

// ResolvePackage resolves the package at path using the current context.
func (c *Context) ResolvePackage(path string) (*Package, error) {
	return loadPackage(c, nil, path)
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

// Destroy removes the temporary working files of this context.
func (c *Context) Destroy() error {
	log.Debugf("removing work directory: %v", c.workdir)
	return os.RemoveAll(c.workdir)
}

// ctxString returns a string representation of the unique properties
// of the context.
func (c *Context) ctxString() string {
	v := []string{
		c.gotargetos,
		c.gotargetarch,
	}
	v = append(v, c.buildtags...)
	return strings.Join(v, "-")
}

func run(dir string, env []string, command string, args ...string) error {
	var buf bytes.Buffer
	err := runOut(&buf, dir, env, command, args...)
	if err != nil {
		return fmt.Errorf("# %s %s: %v\n%s", command, strings.Join(args, " "), err, buf.String())
	}
	return nil
}

func runOut(output io.Writer, dir string, env []string, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	cmd.Stdout = output
	cmd.Stderr = os.Stderr
	cmd.Env = mergeEnvLists(env, envForDir(cmd.Dir))
	log.Debugf("cd %s; %s", cmd.Dir, cmd.Args)
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

// shouldignore tests if the package should be ignored.
func (c *Context) shouldignore(p string) bool {
	if c.isCrossCompile() {
		return p == "C" || p == "unsafe"
	}
	return stdlib[p]
}

func (c *Context) isCrossCompile() bool {
	return c.gohostos != c.gotargetos || c.gohostarch != c.gotargetarch
}

func matchPackages(c *Context, pattern string) []string {
	log.Debugf("matchPackages: %v %v", c.srcdirs[0].Root, pattern)
	match := func(string) bool { return true }
	treeCanMatch := func(string) bool { return true }
	if pattern != "all" && pattern != "std" {
		match = matchPattern(pattern)
		treeCanMatch = treeCanMatchPattern(pattern)
	}

	var pkgs []string

	for _, dir := range c.srcdirs[:1] {
		src := filepath.Clean(dir.Root) + string(filepath.Separator)
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
