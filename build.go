package gb

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/constabulary/gb/log"
)

// Build builds each of pkgs in succession. If pkg is a command, then the results of build include
// linking the final binary into pkg.Context.Bindir().
func Build(pkgs ...*Package) error {
	build, err := BuildPackages(pkgs...)
	if err != nil {
		return err
	}
	return ExecuteConcurrent(build, runtime.NumCPU())
}

// BuildPackages produces a tree of *Actions that can be executed to build
// a *Package.
// BuildPackages walks the tree of *Packages and returns a corresponding
// tree of *Actions representing the steps required to build *Package
// and any of its dependencies
func BuildPackages(pkgs ...*Package) (*Action, error) {
	if len(pkgs) < 1 {
		return nil, fmt.Errorf("no packages supplied")
	}

	targets := make(map[string]*Action) // maps package importpath to build action

	names := func(pkgs []*Package) []string {
		var names []string
		for _, pkg := range pkgs {
			names = append(names, pkg.ImportPath)
		}
		return names
	}

	// create top level build action to unify all packages
	t0 := time.Now()
	build := Action{
		Name: fmt.Sprintf("build: %s", strings.Join(names(pkgs), ",")),
		Task: TaskFn(func() error {
			log.Debugf("build duration: %v %v", time.Since(t0), pkgs[0].Statistics.String())
			return nil
		}),
	}

	for _, pkg := range pkgs {
		if len(pkg.GoFiles)+len(pkg.CgoFiles) == 0 {
			log.Debugf("skipping %v: no go files", pkg.ImportPath)
			continue
		}
		a, err := BuildPackage(targets, pkg)
		if err != nil {
			return nil, err
		}
		if a == nil {
			// nothing to do
			continue
		}
		build.Deps = append(build.Deps, a)
	}
	return &build, nil
}

// BuildPackage returns an Action representing the steps required to
// build this package.
func BuildPackage(targets map[string]*Action, pkg *Package) (*Action, error) {

	// if this action is already present in the map, return it
	// rather than creating a new action.
	if a, ok := targets[pkg.ImportPath]; ok {
		return a, nil
	}

	// step 0. are we stale ?
	// if this package is not stale, then by definition none of its
	// dependencies are stale, so ignore this whole tree.
	if !pkg.Stale {
		return nil, nil
	}

	// step 1. build dependencies
	deps, err := BuildDependencies(targets, pkg)
	if err != nil {
		return nil, err
	}

	// step 2. build this package
	build, err := Compile(pkg, deps...)
	if err != nil {
		return nil, err
	}

	if build == nil {
		panic("build action was nil") // shouldn't happen
	}

	// record the final action as the action that represents
	// building this package.
	targets[pkg.ImportPath] = build
	return build, nil
}

// Compile returns an Action representing the steps required to compile this package.
func Compile(pkg *Package, deps ...*Action) (*Action, error) {
	var gofiles []string
	gofiles = append(gofiles, pkg.GoFiles...)

	// step 1. are there any .c files that we have to run cgo on ?
	var ofiles []string // additional ofiles to pack
	if len(pkg.CgoFiles) > 0 {
		cgoACTION, cgoOFILES, cgoGOFILES, err := cgo(pkg)
		if err != nil {
			return nil, err
		}

		gofiles = append(gofiles, cgoGOFILES...)
		ofiles = append(ofiles, cgoOFILES...)
		deps = append(deps, cgoACTION)
	}

	if len(gofiles) == 0 {
		return nil, fmt.Errorf("compile %q: no go files supplied", pkg.ImportPath)
	}

	// step 2. compile all the go files for this package, including pkg.CgoFiles
	compile := Action{
		Name: fmt.Sprintf("compile: %s", pkg.ImportPath),
		Deps: deps,
		Task: TaskFn(func() error {
			return gc(pkg, gofiles)
		}),
	}

	// step 3. are there any .s files to assemble.
	var assemble []*Action
	for _, sfile := range pkg.SFiles {
		sfile := sfile
		ofile := filepath.Join(pkg.Workdir(), pkg.ImportPath, stripext(sfile)+".6")
		assemble = append(assemble, &Action{
			Name: fmt.Sprintf("asm: %s/%s", pkg.ImportPath, sfile),
			Task: TaskFn(func() error {
				t0 := time.Now()
				err := pkg.tc.Asm(pkg, pkg.Dir, ofile, filepath.Join(pkg.Dir, sfile))
				pkg.Record("asm", time.Since(t0))
				return err
			}),
			// asm depends on compile because compile will generate the local go_asm.h
			Deps: []*Action{&compile},
		})
		ofiles = append(ofiles, ofile)
	}

	build := &compile

	// Do we need to pack ? Yes, replace build action with pack.
	if len(ofiles) > 0 {
		pack := Action{
			Name: fmt.Sprintf("pack: %s", pkg.ImportPath),
			Deps: []*Action{
				&compile,
			},
			Task: TaskFn(func() error {
				// collect .o files, ofiles always starts with the gc compiled object.
				// TODO(dfc) objfile(pkg) should already be at the top of this set
				ofiles = append(
					[]string{objfile(pkg)},
					ofiles...,
				)

				// pack
				t0 := time.Now()
				err := pkg.tc.Pack(pkg, ofiles...)
				pkg.Record("pack", time.Since(t0))
				return err
			}),
		}
		pack.Deps = append(pack.Deps, assemble...)
		build = &pack
	}

	// should this package be cached
	// TODO(dfc) pkg.SkipInstall should become Install
	if !pkg.SkipInstall && pkg.Scope != "test" {
		install := Action{
			Name: fmt.Sprintf("install: %s", pkg.ImportPath),
			Deps: []*Action{
				build,
			},
			Task: TaskFn(func() error {
				return copyfile(pkgfile(pkg), objfile(pkg))
			}),
		}
		build = &install
	}

	// if this is a main package, add a link stage
	if pkg.isMain() {
		link := Action{
			Name: fmt.Sprintf("link: %s", pkg.ImportPath),
			Deps: []*Action{build},
			Task: TaskFn(func() error {
				return link(pkg)
			}),
		}
		build = &link
	}
	return build, nil
}

// BuildDependencies returns a slice of Actions representing the steps required
// to build all dependant packages of this package.
func BuildDependencies(targets map[string]*Action, pkg *Package) ([]*Action, error) {
	var deps []*Action
	for _, i := range pkg.Imports() {
		a, err := BuildPackage(targets, i)
		if err != nil {
			return nil, err
		}
		if a == nil {
			// no action required for this Package
			continue
		}
		deps = append(deps, a)
	}
	return deps, nil
}

func gc(pkg *Package, gofiles []string) error {
	t0 := time.Now()
	if pkg.Scope != "test" {
		// only log compilation message if not in test scope
		log.Infof(pkg.ImportPath)
	}
	includes := pkg.IncludePaths()
	importpath := pkg.ImportPath
	if pkg.Scope == "test" && pkg.ExtraIncludes != "" {
		// TODO(dfc) gross
		includes = append([]string{pkg.ExtraIncludes}, includes...)
	}
	for i := range gofiles {
		if filepath.IsAbs(gofiles[i]) {
			// terrible hack for cgo files which come with an absolute path
			continue
		}
		fullpath := filepath.Join(pkg.Dir, gofiles[i])
		path, err := filepath.Rel(pkg.Dir, fullpath)
		if err == nil {
			gofiles[i] = path
		} else {
			gofiles[i] = fullpath
		}
	}
	err := pkg.tc.Gc(pkg, includes, importpath, pkg.Dir, objfile(pkg), gofiles)
	pkg.Record("gc", time.Since(t0))
	return err
}

func link(pkg *Package) error {
	t0 := time.Now()
	target := pkg.Binfile()
	if err := mkdir(filepath.Dir(target)); err != nil {
		return err
	}

	includes := pkg.IncludePaths()
	if pkg.Scope == "test" && pkg.ExtraIncludes != "" {
		// TODO(dfc) gross
		includes = append([]string{pkg.ExtraIncludes}, includes...)
	}
	err := pkg.tc.Ld(pkg, includes, target, objfile(pkg))
	pkg.Record("link", time.Since(t0))
	return err
}

// Workdir returns the working directory for a package.
func Workdir(pkg *Package) string {
	switch pkg.Scope {
	case "test":
		ip := strings.TrimSuffix(filepath.FromSlash(pkg.ImportPath), "_test")
		return filepath.Join(pkg.Workdir(), ip, "_test", filepath.Dir(filepath.FromSlash(pkg.ImportPath)))
	default:
		return filepath.Join(pkg.Workdir(), filepath.Dir(filepath.FromSlash(pkg.ImportPath)))
	}
}

// objfile returns the name of the object file for this package
func objfile(pkg *Package) string {
	return filepath.Join(Workdir(pkg), objname(pkg))
}

func objname(pkg *Package) string {
	if pkg.isMain() {
		return filepath.Join(filepath.Base(filepath.FromSlash(pkg.ImportPath)), "main.a")
	}
	return filepath.Base(filepath.FromSlash(pkg.ImportPath)) + ".a"
}

func pkgname(pkg *Package) string {
	switch pkg.Scope {
	case "test":
		return filepath.Base(filepath.FromSlash(pkg.ImportPath))
	default:
		if pkg.Name == "main" {
			return filepath.Base(filepath.FromSlash(pkg.ImportPath))
		}
		return pkg.Name
	}
}

func binname(pkg *Package) string {
	switch {
	case pkg.Scope == "test":
		return pkg.Name + ".test"
	case pkg.Name == "main":
		return filepath.Base(filepath.FromSlash(pkg.ImportPath))
	default:
		panic("binname called with non main package: " + pkg.ImportPath)
	}
}
