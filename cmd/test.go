package cmd

import (
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/log"
)

// Test returns a Target representing the result of compiling the
// package pkg, and its dependencies, and linking it with the
// test runner.
func Test(flags []string, pkgs ...*gb.Package) error {
	test, err := TestPackages(flags, pkgs...)
	if err != nil {
		return err
	}
	return gb.Execute(test)
}

// TestPackages produces a graph of Actions that when executed build
// and test the supplied packages.
func TestPackages(flags []string, pkgs ...*gb.Package) (*gb.Action, error) {
	if len(pkgs) < 1 {
		return nil, fmt.Errorf("no test packages provided")
	}
	targets := make(map[string]*gb.Action) // maps package import paths to their test run action

	names := func(pkgs []*gb.Package) []string {
		var names []string
		for _, pkg := range pkgs {
			names = append(names, pkg.ImportPath)
		}
		return names
	}

	// create top level test action to root all test actions
	t0 := time.Now()
	test := gb.Action{
		Name: fmt.Sprintf("test: %s", strings.Join(names(pkgs), ",")),
		Task: gb.TaskFn(func() error {
			log.Debugf("test duration: %v %v", time.Since(t0), pkgs[0].Statistics.String())
			return nil
		}),
	}

	for _, pkg := range pkgs {
		a, err := TestPackage(targets, pkg, flags)
		if err != nil {
			return nil, err
		}
		if a == nil {
			// nothing to do ?? not even a test action ?
			continue
		}
		test.Deps = append(test.Deps, a)
	}
	return &test, nil
}

// TestPackage returns an Action representing the steps required to build
// and test this Package.
func TestPackage(targets map[string]*gb.Action, pkg *gb.Package, flags []string) (*gb.Action, error) {
	var gofiles []string
	gofiles = append(gofiles, pkg.GoFiles...)
	gofiles = append(gofiles, pkg.TestGoFiles...)

	var cgofiles []string
	cgofiles = append(cgofiles, pkg.CgoFiles...)

	var imports []string
	imports = append(imports, pkg.Package.Imports...)
	imports = append(imports, pkg.Package.TestImports...)

	name := pkg.Name
	if name == "main" {
		// rename the main package to its package name for testing.
		name = filepath.Base(filepath.FromSlash(pkg.ImportPath))
	}

	// internal tests
	testpkg := gb.NewPackage(pkg.Context, &build.Package{
		Name:       name,
		ImportPath: pkg.ImportPath,
		Dir:        pkg.Dir,
		SrcRoot:    pkg.SrcRoot,

		GoFiles:      gofiles,
		CFiles:       pkg.CFiles,
		CgoFiles:     cgofiles,
		TestGoFiles:  pkg.TestGoFiles,  // passed directly to buildTestMain
		XTestGoFiles: pkg.XTestGoFiles, // passed directly to buildTestMain

		CgoCFLAGS:    pkg.CgoCFLAGS,
		CgoCPPFLAGS:  pkg.CgoCPPFLAGS,
		CgoCXXFLAGS:  pkg.CgoCXXFLAGS,
		CgoLDFLAGS:   pkg.CgoLDFLAGS,
		CgoPkgConfig: pkg.CgoPkgConfig,

		Imports: imports,
	})
	testpkg.Scope = "test"
	testpkg.Stale = true

	// build dependencies
	deps, err := gb.BuildDependencies(targets, testpkg)
	if err != nil {
		return nil, err
	}

	// only build the internal test if there is Go source or
	// internal test files.
	var testobj *gb.Action
	if len(testpkg.GoFiles)+len(testpkg.CgoFiles)+len(testpkg.TestGoFiles) > 0 {
		var err error
		testobj, err = gb.Compile(testpkg, deps...)
		if err != nil {
			return nil, err
		}
	}

	// external tests
	if len(pkg.XTestGoFiles) > 0 {
		xtestpkg := gb.NewPackage(pkg.Context, &build.Package{
			Name:       name,
			ImportPath: pkg.ImportPath + "_test",
			Dir:        pkg.Dir,
			GoFiles:    pkg.XTestGoFiles,
			Imports:    pkg.XTestImports,
		})
		// build external test dependencies
		deps, err := gb.BuildDependencies(targets, xtestpkg)
		if err != nil {
			return nil, err
		}
		xtestpkg.Scope = "test"
		xtestpkg.Stale = true
		xtestpkg.ExtraIncludes = filepath.Join(pkg.Workdir(), filepath.FromSlash(pkg.ImportPath), "_test")

		// if there is an internal test object, add it as a dependency.
		if testobj != nil {
			deps = append(deps, testobj)
		}
		testobj, err = gb.Compile(xtestpkg, deps...)
		if err != nil {
			return nil, err
		}
	}

	testmainpkg, err := buildTestMain(testpkg)
	if err != nil {
		return nil, err
	}
	testmain, err := gb.Compile(testmainpkg, testobj)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(testmainpkg.Binfile(), flags...)
	cmd.Dir = pkg.Dir // tests run in the original source directory
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Debugf("scheduling run of %v", cmd.Args)
	return &gb.Action{
		Name: fmt.Sprintf("run: %s", cmd.Args),
		Deps: []*gb.Action{testmain},
		Task: gb.TaskFn(func() error {
			return cmd.Run()
		}),
	}, nil
}

func buildTestMain(pkg *gb.Package) (*gb.Package, error) {
	if pkg.Scope != "test" {
		return nil, fmt.Errorf("package %q is not test scoped", pkg.Name)
	}
	dir := gb.Workdir(pkg)
	if err := mkdir(dir); err != nil {
		return nil, fmt.Errorf("buildTestmain: %v", err)
	}
	tests, err := loadTestFuncs(pkg.Package)
	if err != nil {
		return nil, err
	}
	if len(pkg.Package.XTestGoFiles) > 0 {
		// if there are external tests ensure that we import the
		// test package into the final binary for side effects.
		tests.ImportXtest = true
	}
	if err := writeTestmain(filepath.Join(dir, "_testmain.go"), tests); err != nil {
		return nil, err
	}
	testmain := gb.NewPackage(pkg.Context, &build.Package{
		Name:       pkg.Name,
		ImportPath: path.Join(pkg.ImportPath, "testmain"),
		Dir:        dir,
		SrcRoot:    pkg.SrcRoot,

		GoFiles: []string{"_testmain.go"},

		Imports: pkg.Package.Imports,
	})
	testmain.Scope = "test"
	testmain.ExtraIncludes = filepath.Join(pkg.Workdir(), filepath.FromSlash(pkg.ImportPath), "_test")
	return testmain, nil
}
