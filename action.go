package gb

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Actions and Tasks.
//
// Actions and Tasks allow gb to separate the role of describing the
// order in which work will be done, from describing that work itself.
// Actions are the former, they describe the graph of dependencies
// between actions, and thus the work to be done. By traversing the action
// graph, we can do the work, execute the Tasks in a sane order.
//
// Tasks describe the work to be done, without being concerned with
// the order in which the work is done -- that is up to the code that
// places Tasks into actions. Tasks also know more intimate details about
// filesystems, processes, file lists, etc that Actions do not.
//
// Action graphs (they are not strictly trees as branchs converge on base actions)
// contain only work to be performed, there are no Actions with empty Tasks
// or Tasks which do no work.
//
// Actions are executed by Executors, but can also be transformed, mutated,
// or even graphed.

// An Action describes a task to be performed and a set
// of Actions that task depends on.
type Action struct {

	// Name describes the action.
	Name string

	// Deps identifies the Actions that this Action depends.
	Deps []*Action

	// Task identifies the that this action represents.
	Task
}

// Task represents some work to be performed. It contains a single method
// Run, which is expected to be executed at most once.
type Task interface {

	// Run will initiate the work that this task represents and
	// block until the work is complete.
	Run() error
}

// TaskFn is a Task that can execute itself.
type TaskFn func() error

func (fn TaskFn) Run() error { return fn() }

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

	asmofile := func(sfile string) string {
		return filepath.Join(pkg.Workdir(), pkg.ImportPath, stripext(sfile)+".6")
	}

	var assemble []*Action
	for _, sfile := range pkg.SFiles {
		sfile := sfile
		assemble = append(assemble, &Action{
			Name: fmt.Sprintf("asm: %s/%s", pkg.ImportPath, sfile),
			Task: TaskFn(func() error {
				ofile := asmofile(sfile)
				t0 := time.Now()
				err := pkg.tc.Asm(pkg, pkg.Dir, ofile, filepath.Join(pkg.Dir, sfile))
				pkg.Record("asm", time.Since(t0))
				return err
			}),
		})
	}

	var gofiles []string
	gofiles = append(gofiles, pkg.GoFiles...)

	gc := gc{
		pkg:     pkg,
		gofiles: gofiles,
	}

	compile := Action{
		Name: fmt.Sprintf("compile: %s", pkg.ImportPath),
		Deps: deps,
		Task: TaskFn(gc.compile),
	}

	build := &compile

	// Do we need to pack ? Yes, replace build action with pack.
	if len(assemble) > 0 {
		pack := Action{
			Name: fmt.Sprintf("pack: %s", pkg.ImportPath),
			Deps: []*Action{
				&compile,
			},
			Task: TaskFn(func() error {
				// collect .o files, ofiles always starts with the gc compiled object.
				ofiles := []string{
					objfile(pkg),
				}

				// collect .o files for assemble
				for _, sfile := range pkg.SFiles {
					ofiles = append(ofiles, asmofile(sfile))
				}

				// TODO(dfc) collect cgo .o files

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
		ld := ld{
			pkg:   pkg,
			afile: &gc,
		}
		link := Action{
			Name: fmt.Sprintf("link: %s", pkg.ImportPath),
			Deps: []*Action{build},
			Task: TaskFn(ld.link),
		}
		build = &link
	}
	targets[pkg.ImportPath] = build
	return build, nil
}
