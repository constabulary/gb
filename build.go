package gb

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

func BuildDependencies(targets map[string]PkgTarget, pkg *Package) []Target { panic("unimplemented") }

// Build builds each of pkgs in succession. If pkg is a command, then the results of build include
// linking the final binary into pkg.Context.Bindir().||./
func Build(pkgs ...*Package) error {
	build, err := BuildAction(pkgs...)
	if err != nil {
		return err
	}
	return Execute(build)
}

// BuildAction produces a tree of *Actions that can be executed to build
// a *Package.
// BuildAction walks the tree of *Packages and returns a corresponding
// tree of *Actions representing the steps required to build *Package
// and any of its dependencies
func BuildAction(pkgs ...*Package) (*Action, error) {
	targets := make(map[string]*Action) // maps package importpath ot Action name

	names := func(pkgs []*Package) []string {
		var names []string
		for _, pkg := range pkgs {
			names = append(names, pkg.ImportPath)
		}
		return names
	}

	// create top level build action to unify all packages
	build := Action{
		Name: fmt.Sprintf("build: %s", strings.Join(names(pkgs), ",")),
		Task: TaskFn(func() error {
			fmt.Println("built")
			return nil
		}),
	}
	for _, pkg := range pkgs {
		a, err := buildAction0(targets, pkg)
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

func buildAction0(targets map[string]*Action, pkg *Package) (*Action, error) {

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

	// step 1. walk dependencies
	var deps []*Action
	for _, i := range pkg.Imports() {
		a, err := buildAction0(targets, i)
		if err != nil {
			return nil, err
		}
		if a == nil {
			// no action required for this Package
			continue
		}
		deps = append(deps, a)
	}

	// step 2. create a tree of tasks and actions for building this package.

	// step 2a. are there any .s files to assemble.

	var assemble []*Action
	var ofiles []string // additional ofiles to pack
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
		})
		ofiles = append(ofiles, ofile)
	}

	var gofiles []string
	gofiles = append(gofiles, pkg.GoFiles...)

	// step 2b. are there any .c files that we have to run cgo on ?

	if len(pkg.CgoFiles) > 0 {
		cgoACTION, cgoOFILES, cgoGOFILES, err := cgo(pkg)
		if err != nil {
			return nil, err
		}

		gofiles = append(gofiles, cgoGOFILES...)
		ofiles = append(ofiles, cgoOFILES...)
		deps = append(deps, cgoACTION)
	}

	// step 2c. compile all the go files for this package, including pkg.CgoFiles

	compile := Action{
		Name: fmt.Sprintf("compile: %s", pkg.ImportPath),
		Deps: deps,
		Task: TaskFn(func() error {
			return Compile(pkg, gofiles)
		}),
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
				return Link(pkg)
			}),
		}
		build = &link
	}

	// record the final action as the action that represents
	// building this package.
	targets[pkg.ImportPath] = build
	return build, nil
}

// ObjTarget represents a compiled Go object (.5, .6, etc)
type ObjTarget interface {
	Target

	// Objfile is the name of the file that is produced if the target is successful.
	Objfile() string
}

func Compile(pkg *Package, gofiles []string) error {
	t0 := time.Now()
	if pkg.Scope != "test" {
		// only log compilation message if not in test scope
		Infof(pkg.ImportPath)
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
		path, err := filepath.Rel(pkg.Projectdir(), fullpath)
		if err == nil {
			gofiles[i] = path
		} else {
			gofiles[i] = fullpath
		}
	}
	err := pkg.tc.Gc(pkg, includes, importpath, pkg.Projectdir(), objfile(pkg), gofiles, pkg.Complete())
	pkg.Record("compile", time.Since(t0))
	return err
}

type objpkgtarget interface {
	ObjTarget
	Pkgfile() string // implements PkgTarget
}

// PkgTarget represents a Target that produces a pkg (.a) file.
type PkgTarget interface {
	Target

	// Pkgfile returns the name of the file that is produced by the Target if successful.
	Pkgfile() string
}

func Link(pkg *Package) error {
	t0 := time.Now()
	target := pkg.Binfile()
	if err := mkdir(filepath.Dir(target)); err != nil {
		return err
	}

	includes := pkg.IncludePaths()
	if pkg.Scope == "test" && pkg.ExtraIncludes != "" {
		// TODO(dfc) gross
		includes = append([]string{pkg.ExtraIncludes}, includes...)
		target += ".test"
	}
	err := pkg.tc.Ld(pkg, includes, target, objfile(pkg))
	pkg.Record("link", time.Since(t0))
	return err
}

// objfile returns the name of the object file for this package
func objfile(pkg *Package) string {
	return filepath.Join(pkg.Objdir(), objname(pkg))
}

func objname(pkg *Package) string {
	switch pkg.Name {
	case "main":
		return filepath.Join(filepath.Base(filepath.FromSlash(pkg.ImportPath)), "main.a")
	default:
		return filepath.Base(filepath.FromSlash(pkg.ImportPath)) + ".a"
	}
}

func pkgname(pkg *Package) string {
	if pkg.isMain() {
		return filepath.Base(filepath.FromSlash(pkg.ImportPath))
	}
	return pkg.Name
}

// Binfile returns the destination of the compiled target of this command.
// TODO(dfc) this should be Target.
func (pkg *Package) Binfile() string {
	// TODO(dfc) should have a check for package main, or should be merged in to objfile.
	var target string
	switch pkg.Scope {
	case "test":
		target = filepath.Join(pkg.Workdir(), filepath.FromSlash(pkg.ImportPath), "_test", binname(pkg))
	default:
		target = filepath.Join(pkg.Bindir(), binname(pkg))
	}
	if pkg.GOOS == "windows" {
		target += ".exe"
	}
	return target
}

func binname(pkg *Package) string {
	switch {
	case pkg.Name == "main":
		return filepath.Base(filepath.FromSlash(pkg.ImportPath))
	case pkg.Scope == "test":
		return pkg.Name
	default:
		panic("binname called with non main package: " + pkg.ImportPath)
	}
}
