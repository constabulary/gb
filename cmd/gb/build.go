package main

import (
	"flag"
	"os"
	"path/filepath"
	"runtime"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
	"github.com/constabulary/gb/internal/importer"
	"github.com/pkg/errors"
)

func init() {
	registerCommand(BuildCmd)
}

var (
	// build flags

	// should we build all packages in this project.
	// defaults to true when build is invoked from the project root.
	A bool

	// should we perform a release build +release tag ?
	// defaults to false, +debug.
	R bool

	// force rebuild of packages
	F bool

	// skip caching of packages
	FF bool

	// enable race runtime
	race bool

	ldflags, gcflags []string

	P int // number of executors to run in parallel

	dotfile string // path to dot output file

	buildtags []string
)

func addBuildFlags(fs *flag.FlagSet) {
	// TODO(dfc) this should accept a *gb.Context
	fs.BoolVar(&A, "a", false, "build all packages in this project")
	fs.BoolVar(&R, "r", false, "perform a release build")
	fs.BoolVar(&F, "f", false, "rebuild up-to-date packages")
	fs.BoolVar(&FF, "F", false, "do not cache built packages")
	fs.BoolVar(&race, "race", false, "enable race detector")
	fs.IntVar(&P, "P", runtime.NumCPU(), "number of parallel jobs")
	fs.Var((*stringsFlag)(&ldflags), "ldflags", "flags passed to the linker")
	fs.Var((*stringsFlag)(&gcflags), "gcflags", "flags passed to the compiler")
	fs.StringVar(&dotfile, "dotfile", "", "path to dot output file")
	fs.Var((*stringsFlag)(&buildtags), "tags", "")
}

var BuildCmd = &cmd.Command{
	Name:      "build",
	Short:     "build a package",
	UsageLine: "build [build flags] [packages]",
	Long: `
Build compiles the packages named by the import paths, along with their
dependencies.

Flags:

	-f
		ignore cached packages if present, new packages built will overwrite
		any cached packages. This effectively disables incremental
		compilation.
	-F
		do not cache packages, cached packages will still be used for
		incremental compilation. -f -F is advised to disable the package
		caching system.
	-P
		The number of build jobs to run in parallel, including test execution.
		By default this is the number of CPUs visible to gb.
	-R
		sets the base of the project root search path from the current working
		directory to the value supplied. Effectively gb changes working
		directory to this path before searching for the project root.
	-dotfile
		if provided, gb will output a dot formatted file of the build steps to
		be performed.
	-ldflags 'flag list'
		arguments to pass on each linker invocation.
	-gcflags 'arg list'
		arguments to pass on each compile invocation.
        -race
                enable data race detection.
                Supported only on linux/amd64, freebsd/amd64, darwin/amd64 and windows/amd64.
	-tags 'tag list'
		additional build tags.

The list flags accept a space-separated list of strings. To embed spaces in an
element in the list, surround it with either single or double quotes.

For more about where packages and binaries are installed, run 'gb help project'.
`,
	Run: func(ctx *gb.Context, args []string) error {
		// TODO(dfc) run should take a *gb.Context not a *gb.Project
		ctx.Force = F
		ctx.Install = !FF

		pkgs, err := resolveRootPackages(ctx, args...)
		if err != nil {
			return err
		}

		build, err := gb.BuildPackages(pkgs...)
		if err != nil {
			return err
		}

		if dotfile != "" {
			f, err := os.Create(dotfile)
			if err != nil {
				return err
			}
			defer f.Close()
			printActions(f, build)
		}

		startSigHandlers()
		return gb.ExecuteConcurrent(build, P, interrupted)
	},
	AddFlags: addBuildFlags,
}

// Resolver resolves packages.
type Resolver interface {
	ResolvePackage(path string) (*gb.Package, error)
}

// resolveRootPackages resolves import paths into packages.
// Only packages which exist inside $PROJECT/src are elegable to be
// roots to build or test. Other import paths are discarded.
func resolveRootPackages(r Resolver, paths ...string) ([]*gb.Package, error) {
	var pkgs []*gb.Package
	for _, path := range paths {
		pkg, err := r.ResolvePackage(path)
		if _, nogo := errors.Cause(err).(*importer.NoGoError); nogo {
			// if the package is empty, to no Go files are in scope
			// ignore it.
			// TODO(dfc) ResolvePackage should return an empty *Package
			// and build/test should ignore them.
			continue
		}
		if err != nil {
			return pkgs, errors.Wrapf(err, "failed to resolve import path %q", path)
		}
		if pkg.SrcRoot == filepath.Join(runtime.GOROOT(), "src") {
			// skip package roots that are not part of this project.
			// TODO(dfc) should gb return an error here?
			continue
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs, nil
}
