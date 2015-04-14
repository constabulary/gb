package gb

import (
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	gofiles = append(gofiles, pkg.p.GoFiles...)
	gofiles = append(gofiles, pkg.p.TestGoFiles...)

	var cgofiles []string
	cgofiles = append(cgofiles, pkg.p.CgoFiles...)

	var imports []string
	imports = append(imports, pkg.p.Imports...)
	imports = append(imports, pkg.p.TestImports...)

	// build dependencies
	var deps []Target
	for _, dep := range imports {
		pkg := pkg.ctx.ResolvePackage(dep)
		deps = append(deps, Build(pkg))
	}

	testpkg := &build.Package{
		Name:       pkg.Name(),
		ImportPath: pkg.ImportPath,
		// Srcdir:     pkg.Srcdir,

		GoFiles:     gofiles,
		CgoFiles:    cgofiles,
		TestGoFiles: pkg.p.TestGoFiles, // passed directly to buildTestMain

		Imports: imports,
	}

	test := newPackage(pkg.ctx, testpkg)
	compile := Compile(test, deps...)
	buildtest := buildTest(test, compile)
	runtest := runTest(test, buildtest)
	return runtest
}

type buildTestTarget struct {
	target
	pkg *Package
}

func (t *buildTestTarget) build() error {
	if err := buildTestMain(t.pkg); err != nil {
		return err
	}
	gc := Gc(t.pkg, []string{"_testmain.go"})
	pack := Pack(t.pkg, gc)
	return Ld(t.pkg, pack).Result()
}

func buildTestMain(pkg *Package) error {
	return writeTestmain(filepath.Join(objdir(pkg), "_testmain.go"), pkg.p)
}

func buildTest(pkg *Package, deps ...Target) Target {
	t := buildTestTarget{
		pkg: pkg,
	}
	t.target = newTarget(t.build, deps...)
	return &t
}

type runTestTarget struct {
	target
	pkg *Package
}

func (t *runTestTarget) runTest() error {
	Infof("test %q", t.pkg.ImportPath)
	cmd := exec.Command(filepath.Join(objdir(t.pkg), t.pkg.Name()+".test"))
	cmd.Dir = t.pkg.p.Dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	Debugf("cd %s; %s", cmd.Dir, strings.Join(cmd.Args, " "))
	return cmd.Run()
}

func runTest(pkg *Package, deps ...Target) Target {
	t := runTestTarget{
		pkg: pkg,
	}
	t.target = newTarget(t.runTest, deps...)
	return &t
}

// testobjdir returns the destination for test object files compiled for this Package.
func testobjdir(pkg *Package) string {
	return filepath.Join(pkg.ctx.workdir, filepath.FromSlash(pkg.p.ImportPath), "_test")
}
