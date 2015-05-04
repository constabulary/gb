package gb

import (
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"path"
	"path/filepath"
)

// Test returns a Target representing the result of compiling the
// package pkg, and its dependencies, and linking it with the
// test runner.
func Test(pkgs ...*Package) error {
	targets := make(map[string]PkgTarget)
	roots := make([]Target, 0, len(pkgs))
	for _, pkg := range pkgs {
		// commands are built as packages for testing.
		target := testPackage(targets, pkg)
		roots = append(roots, target)
	}
	for _, root := range roots {
		if err := root.Result(); err != nil {
			return err
		}
	}
	return nil
}

func testPackage(targets map[string]PkgTarget, pkg *Package) Target {
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

	testpkg := newPackage(pkg.ctx, &build.Package{
		Name:       name,
		ImportPath: pkg.ImportPath,
		Dir:        pkg.Dir,
		SrcRoot:    pkg.SrcRoot,

		GoFiles:      gofiles,
		CgoFiles:     cgofiles,
		TestGoFiles:  pkg.TestGoFiles,  // passed directly to buildTestMain
		XTestGoFiles: pkg.XTestGoFiles, // passed directly to buildTestMain

		Imports: imports,
	})

	// build dependencies
	deps := buildDependencies(targets, testpkg)
	testpkg.Scope = "test"

	testobj := Compile(testpkg, deps...)

	// external tests
	if len(pkg.XTestGoFiles) > 0 {
		xtestpkg := newPackage(pkg.ctx, &build.Package{
			Name:       pkg.Name,
			ImportPath: pkg.ImportPath + "_test",
			Dir:        pkg.Dir,
			GoFiles:    pkg.XTestGoFiles,
			Imports:    pkg.XTestImports,
		})
		// build external test dependencies
		deps := buildDependencies(targets, xtestpkg)
		xtestpkg.Scope = "test"
		testobj = Compile(xtestpkg, append(deps, testobj)...)
	}

	testmain, err := buildTestMain(testpkg)
	if err != nil {
		return errTarget{err}
	}
	buildmain := Ld(testmain, Compile(testmain, testobj))

	cmd := exec.Command(binfile(testmain) + ".test")
	cmd.Dir = pkg.Dir // tests run in the original source directory
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	Debugf("scheduling run of %v", cmd.Args)
	return Run(cmd, buildmain)
}

func buildTestMain(pkg *Package) (*Package, error) {
	if pkg.Scope != "test" {
		return nil, fmt.Errorf("package %q is not test scoped", pkg.Name)
	}
	dir := objdir(pkg)
	if err := os.MkdirAll(dir, 0755); err != nil {
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
	testmain := newPackage(pkg.ctx, &build.Package{
		Name:       pkg.Name,
		ImportPath: path.Join(pkg.ImportPath, "testmain"),
		Dir:        dir,
		SrcRoot:    pkg.SrcRoot,

		GoFiles: []string{"_testmain.go"},

		Imports: pkg.Package.Imports,
	})
	testmain.Scope = "test"
	testmain.ExtraIncludes = filepath.Join(pkg.ctx.workdir, filepath.FromSlash(pkg.ImportPath), "_test")
	return testmain, nil
}
