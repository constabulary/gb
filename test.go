package gb

import (
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
)

// Test returns a Target representing the result of compiling the
// package pkg, and its dependencies, and linking it with the
// test runner.
func Test(pkg *Package) Target {
	if err := pkg.Result(); err != nil {
		return errTarget{err}
	}
	// commands are built as packages for testing.
	return testPackage(pkg)
}

func testPackage(pkg *Package) Target {
	var gofiles []string
	gofiles = append(gofiles, pkg.p.GoFiles...)
	gofiles = append(gofiles, pkg.p.TestGoFiles...)

	var cgofiles []string
	cgofiles = append(cgofiles, pkg.p.CgoFiles...)

	var imports []string
	imports = append(imports, pkg.p.Imports...)
	imports = append(imports, pkg.p.TestImports...)

	// build dependencies
	deps := buildDependencies(pkg.ctx, imports...)
	Debugf("testing %q: building deps: %v", pkg.Name(), deps)
	testpkg := newPackage(pkg.ctx, &build.Package{
		Name:       pkg.p.Name,
		ImportPath: pkg.ImportPath,
		Dir:        pkg.p.Dir,

		GoFiles:     gofiles,
		CgoFiles:    cgofiles,
		TestGoFiles: pkg.p.TestGoFiles, // passed directly to buildTestMain

		Imports: imports,
	})

	testpkg.Scope = "test"

	testobj := Compile(testpkg, deps...)
	Debugf("building testobj: %v", testobj)
	testmain, err := buildTestMain(testpkg)
	if err != nil {
		return errTarget{err}
	}
	pkgmain := Compile(testmain, testobj)
	buildmain := Ld(testmain, pkgmain.(PkgTarget))

	cmd := exec.Command(filepath.Join(objdir(testmain), testmain.p.Name+".test"))
	cmd.Dir = pkg.p.Dir // tests run in the original source directory
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	Debugf("scheduling run of %v", cmd.Args)
	return Run(cmd, buildmain)
}

func buildTestMain(pkg *Package) (*Package, error) {
	if err := pkg.Result(); err != nil {
		return nil, err
	}
	if pkg.Scope != "test" {
		return nil, fmt.Errorf("package %q is not test scoped", pkg.Name())
	}
	dir := objdir(pkg)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	if err := writeTestmain(filepath.Join(dir, "_testmain.go"), pkg.p); err != nil {
		return nil, err
	}
	testmain := newPackage(pkg.ctx, &build.Package{
		Name:       "main",
		ImportPath: pkg.p.ImportPath,
		Dir:        dir,

		GoFiles: []string{"_testmain.go"},

		Imports: pkg.p.Imports,
	})
	testmain.Scope = "test"
	testmain.ImportPath = "testmain"
	return testmain, nil
}
