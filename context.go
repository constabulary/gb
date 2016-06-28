package gb

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/constabulary/gb/internal/debug"
	"github.com/constabulary/gb/internal/importer"
	"github.com/pkg/errors"
)

// Importer resolves package import paths to *importer.Packages.
type Importer interface {

	// Import attempts to resolve the package import path, path,
	// to an *importer.Package.
	Import(path string) (*importer.Package, error)
}

// Context represents an execution of one or more Targets inside a Project.
type Context struct {
	Project

	importer Importer

	pkgs map[string]*Package // map of package paths to resolved packages

	workdir string

	tc Toolchain

	gohostos, gohostarch     string // GOOS and GOARCH for this host
	gotargetos, gotargetarch string // GOOS and GOARCH for the target

	Statistics

	Force   bool // force rebuild of packages
	Install bool // copy packages into $PROJECT/pkg
	Verbose bool // verbose output
	Nope    bool // command specfic flag, under test it skips the execute action.
	race    bool // race detector requested

	gcflags []string // flags passed to the compiler
	ldflags []string // flags passed to the linker

	linkmode, buildmode string // link and build modes

	buildtags []string // build tags
}

// GOOS configures the Context to use goos as the target os.
func GOOS(goos string) func(*Context) error {
	return func(c *Context) error {
		if goos == "" {
			return fmt.Errorf("GOOS cannot be blank")
		}
		c.gotargetos = goos
		return nil
	}
}

// GOARCH configures the Context to use goarch as the target arch.
func GOARCH(goarch string) func(*Context) error {
	return func(c *Context) error {
		if goarch == "" {
			return fmt.Errorf("GOARCH cannot be blank")
		}
		c.gotargetarch = goarch
		return nil
	}
}

// Tags configured the context to use these additional build tags
func Tags(tags ...string) func(*Context) error {
	return func(c *Context) error {
		c.buildtags = append(c.buildtags, tags...)
		return nil
	}
}

// Gcflags appends flags to the list passed to the compiler.
func Gcflags(flags ...string) func(*Context) error {
	return func(c *Context) error {
		c.gcflags = append(c.gcflags, flags...)
		return nil
	}
}

// Ldflags appends flags to the list passed to the linker.
func Ldflags(flags ...string) func(*Context) error {
	return func(c *Context) error {
		c.ldflags = append(c.ldflags, flags...)
		return nil
	}
}

// WithRace enables the race detector and adds the tag "race" to
// the Context build tags.
func WithRace(c *Context) error {
	c.race = true
	Tags("race")(c)
	Gcflags("-race")(c)
	Ldflags("-race")(c)
	return nil
}

// NewContext returns a new build context from this project.
// By default this context will use the gc toolchain with the
// host's GOOS and GOARCH values.
func NewContext(p Project, opts ...func(*Context) error) (*Context, error) {
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
	workdir, err := ioutil.TempDir("", "gb")
	if err != nil {
		return nil, err
	}

	ctx := Context{
		Project:   p,
		workdir:   workdir,
		buildmode: "exe",
		pkgs:      make(map[string]*Package),
	}

	for _, opt := range append(defaults, opts...) {
		err := opt(&ctx)
		if err != nil {
			return nil, err
		}
	}

	// sort build tags to ensure the ctxSring and Suffix is stable
	sort.Strings(ctx.buildtags)

	ic := importer.Context{
		GOOS:        ctx.gotargetos,
		GOARCH:      ctx.gotargetarch,
		CgoEnabled:  cgoEnabled(ctx.gohostos, ctx.gohostarch, ctx.gotargetos, ctx.gotargetarch),
		ReleaseTags: releaseTags, // from go/build, see gb.go
		BuildTags:   ctx.buildtags,
	}

	i, err := buildImporter(&ic, &ctx)
	if err != nil {
		return nil, err
	}

	ctx.importer = i

	// C and unsafe are fake packages synthesised by the compiler.
	// Insert fake packages into the package cache.
	for _, name := range []string{"C", "unsafe"} {
		pkg, err := newPackage(&ctx, &importer.Package{
			Name:       name,
			ImportPath: name,
			Standard:   true,
			Dir:        name, // fake, but helps diagnostics
		})
		if err != nil {
			return nil, err
		}
		pkg.Stale = false
		ctx.pkgs[pkg.ImportPath] = pkg
	}

	return &ctx, err
}

// IncludePaths returns the include paths visible in this context.
func (c *Context) IncludePaths() []string {
	return []string{
		c.workdir,
		c.Pkgdir(),
	}
}

// NewPackage creates a resolved Package for p.
func (c *Context) NewPackage(p *importer.Package) (*Package, error) {
	pkg, err := newPackage(c, p)
	if err != nil {
		return nil, err
	}
	pkg.Stale = isStale(pkg)
	return pkg, nil
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
	if path == "." {
		return nil, errors.Errorf("%q is not a package", filepath.Join(c.Projectdir(), "src"))
	}
	path, err := relImportPath(filepath.Join(c.Projectdir(), "src"), path)
	if err != nil {
		return nil, err
	}
	if path == "." || path == ".." || strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") {
		return nil, errors.Errorf("import %q: relative import not supported", path)
	}
	return c.loadPackage(nil, path)
}

// loadPackage recursively resolves path as a package. If successful loadPackage
// records the package in the Context's internal package cache.
func (c *Context) loadPackage(stack []string, path string) (*Package, error) {
	if pkg, ok := c.pkgs[path]; ok {
		// already loaded, just return
		return pkg, nil
	}

	p, err := c.importer.Import(path)
	if err != nil {
		return nil, err
	}

	stack = append(stack, p.ImportPath)
	var stale bool
	for i, im := range p.Imports {
		for _, p := range stack {
			if p == im {
				return nil, fmt.Errorf("import cycle detected: %s", strings.Join(append(stack, im), " -> "))
			}
		}
		pkg, err := c.loadPackage(stack, im)
		if err != nil {
			return nil, err
		}

		// update the import path as the import may have been discovered via vendoring.
		p.Imports[i] = pkg.ImportPath
		stale = stale || pkg.Stale
	}

	pkg, err := newPackage(c, p)
	if err != nil {
		return nil, errors.Wrapf(err, "loadPackage(%q)", path)
	}
	pkg.Stale = stale || isStale(pkg)
	c.pkgs[p.ImportPath] = pkg
	return pkg, nil
}

// Destroy removes the temporary working files of this context.
func (c *Context) Destroy() error {
	debug.Debugf("removing work directory: %v", c.workdir)
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

func runOut(output io.Writer, dir string, env []string, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	cmd.Stdout = output
	cmd.Stderr = os.Stderr
	cmd.Env = mergeEnvLists(env, envForDir(cmd.Dir))
	debug.Debugf("cd %s; %s", cmd.Dir, cmd.Args)
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

func (c *Context) isCrossCompile() bool {
	return c.gohostos != c.gotargetos || c.gohostarch != c.gotargetarch
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

func cgoEnabled(gohostos, gohostarch, gotargetos, gotargetarch string) bool {
	switch os.Getenv("CGO_ENABLED") {
	case "1":
		return true
	case "0":
		return false
	default:
		// cgo must be explicitly enabled for cross compilation builds
		if gohostos == gotargetos && gohostarch == gotargetarch {
			switch gotargetos + "/" + gotargetarch {
			case "darwin/386", "darwin/amd64", "darwin/arm", "darwin/arm64":
				return true
			case "dragonfly/amd64":
				return true
			case "freebsd/386", "freebsd/amd64", "freebsd/arm":
				return true
			case "linux/386", "linux/amd64", "linux/arm", "linux/arm64", "linux/ppc64le":
				return true
			case "android/386", "android/amd64", "android/arm":
				return true
			case "netbsd/386", "netbsd/amd64", "netbsd/arm":
				return true
			case "openbsd/386", "openbsd/amd64":
				return true
			case "solaris/amd64":
				return true
			case "windows/386", "windows/amd64":
				return true
			default:
				return false
			}
		}
		return false
	}
}

func buildImporter(ic *importer.Context, ctx *Context) (Importer, error) {
	i, err := addDepfileDeps(ic, ctx)
	if err != nil {
		return nil, err
	}

	// construct importer stack in reverse order, vendor at the bottom, GOROOT on the top.
	i = &_importer{
		Importer: i,
		im: importer.Importer{
			Context: ic,
			Root:    filepath.Join(ctx.Projectdir(), "vendor"),
		},
	}

	i = &srcImporter{
		i,
		importer.Importer{
			Context: ic,
			Root:    ctx.Projectdir(),
		},
	}

	i = &_importer{
		i,
		importer.Importer{
			Context: ic,
			Root:    runtime.GOROOT(),
		},
	}

	i = &fixupImporter{
		Importer: i,
	}

	return i, nil
}
