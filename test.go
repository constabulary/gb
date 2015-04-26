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
	// commands are built as packages for testing.
	return testPackage(pkg)
}

func testPackage(pkg *Package) Target {
	var gofiles []string
	gofiles = append(gofiles, pkg.GoFiles...)
	gofiles = append(gofiles, pkg.TestGoFiles...)

	var cgofiles []string
	cgofiles = append(cgofiles, pkg.CgoFiles...)

	var imports []string
	imports = append(imports, pkg.Package.Imports...)
	imports = append(imports, pkg.Package.TestImports...)

	testpkg := newPackage(pkg.ctx, &build.Package{
		Name:       pkg.Name,
		ImportPath: pkg.ImportPath,
		Dir:        pkg.Dir,
		SrcRoot:    pkg.SrcRoot,

		GoFiles:     gofiles,
		CgoFiles:    cgofiles,
		TestGoFiles: pkg.TestGoFiles, // passed directly to buildTestMain

		Imports: imports,
	})

	// build dependencies
	deps := buildDependencies(testpkg)
	Debugf("testing %q: building deps: %v", pkg.Name, deps)

	testpkg.Scope = "test"

	testobj := Compile(testpkg, deps...)
	Debugf("building testobj: %v", testobj)
	testmain, err := buildTestMain(testpkg)
	if err != nil {
		return errTarget{err}
	}
	buildmain := Ld(testmain, Compile(testmain, testobj))

	cmd := exec.Command(filepath.Join(objdir(testmain), testmain.Name+".test"))
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
	if err := writeTestmain(filepath.Join(dir, "_testmain.go"), pkg.Package); err != nil {
		return nil, err
	}
	testmain := newPackage(pkg.ctx, &build.Package{
		Name:       pkg.Name,
		ImportPath: "testmain",
		Dir:        dir,
		SrcRoot:    pkg.SrcRoot,

		GoFiles: []string{"_testmain.go"},

		Imports: pkg.Package.Imports,
	})
	testmain.Scope = "test"
	testmain.ExtraIncludes = filepath.Join(pkg.ctx.workdir, filepath.FromSlash(pkg.ImportPath), "_test")
	return testmain, nil
}
