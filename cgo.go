package gb

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func cgo(pkg *Package) (*Action, []string, []string, error) {
	switch {
	case gc14:
		return cgo14(pkg)
	case gc15:
		return cgo15(pkg)
	default:
		return nil, nil, nil, fmt.Errorf("unsupported Go version: %v", runtime.Version)
	}
}

// cgo produces a an Action representing the cgo steps
// an ofile representing the result of the cgo steps
// a set of .go files for compilation, and an error.
func cgo14(pkg *Package) (*Action, []string, []string, error) {

	// collect cflags and ldflags from the package
	// the environment, and pkg-config.
	cgoCPPFLAGS, cgoCFLAGS, cgoCXXFLAGS, cgoLDFLAGS := cflags(pkg, false)
	pcCFLAGS, pcLDFLAGS, err := pkgconfig(pkg)
	if err != nil {
		return nil, nil, nil, err
	}
	cgoCFLAGS = append(cgoCFLAGS, pcCFLAGS...)
	cgoLDFLAGS = append(cgoLDFLAGS, pcLDFLAGS...)

	runcgo1 := []*Action{
		&Action{
			Name: "runcgo1: " + pkg.ImportPath,
			Task: TaskFn(func() error {
				return runcgo1(pkg, cgoCFLAGS, cgoLDFLAGS)
			}),
		}}

	workdir := cgoworkdir(pkg)
	defun := filepath.Join(workdir, "_cgo_defun.o")
	rundefun := Action{
		Name: "cc: " + pkg.ImportPath + ": _cgo_defun_c",
		Deps: runcgo1,
		Task: TaskFn(func() error {
			return pkg.tc.Cc(pkg, defun, filepath.Join(workdir, "_cgo_defun.c"))
		}),
	}

	cgofiles := []string{filepath.Join(workdir, "_cgo_gotypes.go")}
	for _, f := range pkg.CgoFiles {
		cgofiles = append(cgofiles, filepath.Join(workdir, stripext(f)+".cgo1.go"))
	}
	cfiles := []string{
		filepath.Join(workdir, "_cgo_main.c"),
		filepath.Join(workdir, "_cgo_export.c"),
	}
	cfiles = append(cfiles, pkg.CFiles...)

	for _, f := range pkg.CgoFiles {
		cfiles = append(cfiles, filepath.Join(workdir, stripext(f)+".cgo2.c"))
	}

	cflags := append(cgoCPPFLAGS, cgoCFLAGS...)
	cxxflags := append(cgoCPPFLAGS, cgoCXXFLAGS...)
	gcc1, ofiles := cgocc(pkg, cflags, cxxflags, cfiles, pkg.CXXFiles, runcgo1...)
	ofile := filepath.Join(filepath.Dir(ofiles[0]), "_cgo_.o")
	gcc2 := Action{
		Name: "gccld: " + pkg.ImportPath + ": _cgo_.o",
		Deps: gcc1,
		Task: TaskFn(func() error {
			return gccld(pkg, cgoCFLAGS, cgoLDFLAGS, ofile, ofiles)
		}),
	}

	dynout := filepath.Join(workdir, "_cgo_import.c")
	imports := stripext(dynout) + ".o"
	runcgo2 := Action{
		Name: "runcgo2: " + pkg.ImportPath,
		Deps: []*Action{&gcc2},
		Task: TaskFn(func() error {
			if err := runcgo2(pkg, dynout, ofile); err != nil {
				return err
			}
			return pkg.tc.Cc(pkg, imports, dynout)
		}),
	}

	allo := filepath.Join(filepath.Dir(ofiles[0]), "_all.o")
	action := Action{
		Name: "rungcc3: " + pkg.ImportPath,
		Deps: []*Action{&runcgo2, &rundefun},
		Task: TaskFn(func() error {
			return rungcc3(pkg, pkg.Dir, allo, ofiles[1:]) // skip _cgo_main.o
		}),
	}
	return &action, []string{defun, imports, allo}, cgofiles, nil
}

// cgo produces a an Action representing the cgo steps
// an ofile representing the result of the cgo steps
// a set of .go files for compilation, and an error.
func cgo15(pkg *Package) (*Action, []string, []string, error) {

	// collect cflags and ldflags from the package
	// the environment, and pkg-config.
	cgoCPPFLAGS, cgoCFLAGS, cgoCXXFLAGS, cgoLDFLAGS := cflags(pkg, false)
	pcCFLAGS, pcLDFLAGS, err := pkgconfig(pkg)
	if err != nil {
		return nil, nil, nil, err
	}
	cgoCFLAGS = append(cgoCFLAGS, pcCFLAGS...)
	cgoLDFLAGS = append(cgoLDFLAGS, pcLDFLAGS...)

	runcgo1 := []*Action{
		&Action{
			Name: "runcgo1: " + pkg.ImportPath,
			Task: TaskFn(func() error {
				return runcgo1(pkg, cgoCFLAGS, cgoLDFLAGS)
			}),
		},
	}

	workdir := cgoworkdir(pkg)
	cgofiles := []string{filepath.Join(workdir, "_cgo_gotypes.go")}
	for _, f := range pkg.CgoFiles {
		cgofiles = append(cgofiles, filepath.Join(workdir, stripext(f)+".cgo1.go"))
	}
	cfiles := []string{
		filepath.Join(workdir, "_cgo_main.c"),
		filepath.Join(workdir, "_cgo_export.c"),
	}
	cfiles = append(cfiles, pkg.CFiles...)

	for _, f := range pkg.CgoFiles {
		cfiles = append(cfiles, filepath.Join(workdir, stripext(f)+".cgo2.c"))
	}

	cflags := append(cgoCPPFLAGS, cgoCFLAGS...)
	cxxflags := append(cgoCPPFLAGS, cgoCXXFLAGS...)
	gcc1, ofiles := cgocc(pkg, cflags, cxxflags, cfiles, pkg.CXXFiles, runcgo1...)
	ofile := filepath.Join(filepath.Dir(ofiles[0]), "_cgo_.o")
	gcc2 := Action{
		Name: "gccld: " + pkg.ImportPath + ": _cgo_.o",
		Deps: gcc1,
		Task: TaskFn(func() error {
			return gccld(pkg, cgoCFLAGS, cgoLDFLAGS, ofile, ofiles)
		}),
	}

	dynout := filepath.Join(workdir, "_cgo_import.go")
	runcgo2 := Action{
		Name: "runcgo2: " + pkg.ImportPath,
		Deps: []*Action{&gcc2},
		Task: TaskFn(func() error {
			return runcgo2(pkg, dynout, ofile)
		}),
	}
	cgofiles = append(cgofiles, dynout)

	allo := filepath.Join(filepath.Dir(ofiles[0]), "_all.o")
	action := Action{
		Name: "rungcc3: " + pkg.ImportPath,
		Deps: []*Action{&runcgo2},
		Task: TaskFn(func() error {
			return rungcc3(pkg, pkg.Dir, allo, ofiles[1:]) // skip _cgo_main.o
		}),
	}

	return &action, []string{allo}, cgofiles, nil
}

// cgocc compiles all .c files.
// TODO(dfc) cxx not done
func cgocc(pkg *Package, cflags, cxxflags, cfiles, cxxfiles []string, deps ...*Action) ([]*Action, []string) {
	workdir := cgoworkdir(pkg)
	var cc []*Action
	var ofiles []string
	for _, cfile := range cfiles {
		cfile := cfile
		ofile := filepath.Join(workdir, stripext(filepath.Base(cfile))+".o")
		ofiles = append(ofiles, ofile)
		cc = append(cc, &Action{
			Name: "rungcc1: " + pkg.ImportPath + ": " + cfile,
			Deps: deps,
			Task: TaskFn(func() error {
				return rungcc1(pkg, cflags, ofile, cfile)
			}),
		})
	}

	for _, cxxfile := range cxxfiles {
		cxxfile := cxxfile
		ofile := filepath.Join(workdir, stripext(filepath.Base(cxxfile))+".o")
		ofiles = append(ofiles, ofile)
		cc = append(cc, &Action{
			Name: "rung++1: " + pkg.ImportPath + ": " + cxxfile,
			Deps: deps,
			Task: TaskFn(func() error {
				return rungpp1(pkg, cxxflags, ofile, cxxfile)
			}),
		})
	}

	return cc, ofiles
}

// rungcc1 invokes gcc to compile cfile into ofile
func rungcc1(pkg *Package, cgoCFLAGS []string, ofile, cfile string) error {
	args := []string{"-g", "-O2",
		"-I", pkg.Dir,
		"-I", filepath.Dir(ofile),
	}
	args = append(args, cgoCFLAGS...)
	args = append(args,
		"-o", ofile,
		"-c", cfile,
	)
	t0 := time.Now()
	gcc := gccCmd(pkg, pkg.Dir)
	err := run(pkg.Dir, nil, gcc[0], append(gcc[1:], args...)...)
	pkg.Record(gcc[0], time.Since(t0))
	return err
}

// rungpp1 invokes g++ to compile cfile into ofile
func rungpp1(pkg *Package, cgoCFLAGS []string, ofile, cfile string) error {
	args := []string{"-g", "-O2",
		"-I", pkg.Dir,
		"-I", filepath.Dir(ofile),
	}
	args = append(args, cgoCFLAGS...)
	args = append(args,
		"-o", ofile,
		"-c", cfile,
	)
	t0 := time.Now()
	gxx := gxxCmd(pkg, pkg.Dir)
	err := run(pkg.Dir, nil, gxx[0], append(gxx[1:], args...)...)
	pkg.Record(gxx[0], time.Since(t0))
	return err
}

// gccld links the o files from rungcc1 into a single _cgo_.o.
func gccld(pkg *Package, cgoCFLAGS, cgoLDFLAGS []string, ofile string, ofiles []string) error {
	args := []string{}
	args = append(args, "-o", ofile)
	args = append(args, ofiles...)
	args = append(args, cgoLDFLAGS...) // this has to go at the end, because reasons!
	t0 := time.Now()

	var cmd []string
	if len(pkg.CXXFiles) > 0 || len(pkg.SwigCXXFiles) > 0 {
		cmd = gxxCmd(pkg, pkg.Dir)
	} else {
		cmd = gccCmd(pkg, pkg.Dir)
	}
	err := run(pkg.Dir, nil, cmd[0], append(cmd[1:], args...)...)
	pkg.Record("gccld", time.Since(t0))
	return err
}

// rungcc3 links all previous ofiles together with libgcc into a single _all.o.
func rungcc3(pkg *Package, dir string, ofile string, ofiles []string) error {
	args := []string{}
	args = append(args, "-o", ofile)
	args = append(args, ofiles...)
	args = append(args, "-Wl,-r", "-nostdlib")
	var cmd []string
	if len(pkg.CXXFiles) > 0 || len(pkg.SwigCXXFiles) > 0 {
		cmd = gxxCmd(pkg, dir)
	} else {
		cmd = gccCmd(pkg, dir)
	}
	if !strings.HasPrefix(cmd[0], "clang") {
		libgcc, err := libgcc(pkg.Context)
		if err != nil {
			return nil
		}
		args = append(args, libgcc)
	}
	t0 := time.Now()
	err := run(dir, nil, cmd[0], append(cmd[1:], args...)...)
	pkg.Record("gcc3", time.Since(t0))
	return err
}

// libgcc returns the value of gcc -print-libgcc-file-name.
func libgcc(ctx *Context) (string, error) {
	args := []string{
		"-print-libgcc-file-name",
	}
	var buf bytes.Buffer
	cmd := gccCmd(&Package{Context: ctx}, "") // TODO(dfc) hack
	err := runOut(&buf, ".", nil, cmd[0], args...)
	return strings.TrimSpace(buf.String()), err
}

func cgotool(ctx *Context) string {
	// TODO(dfc) need ctx.GOROOT method
	return filepath.Join(ctx.Context.GOROOT, "pkg", "tool", ctx.gohostos+"_"+ctx.gohostarch, "cgo")
}

// envList returns the value of the given environment variable broken
// into fields, using the default value when the variable is empty.
func envList(key, def string) []string {
	v := os.Getenv(key)
	if v == "" {
		v = def
	}
	return strings.Fields(v)
}

// Return the flags to use when invoking the C or C++ compilers, or cgo.
func cflags(p *Package, def bool) (cppflags, cflags, cxxflags, ldflags []string) {
	var defaults string
	if def {
		defaults = "-g -O2"
	}

	cppflags = stringList(envList("CGO_CPPFLAGS", ""), p.CgoCPPFLAGS)
	cflags = stringList(envList("CGO_CFLAGS", defaults), p.CgoCFLAGS)
	cxxflags = stringList(envList("CGO_CXXFLAGS", defaults), p.CgoCXXFLAGS)
	ldflags = stringList(envList("CGO_LDFLAGS", defaults), p.CgoLDFLAGS)
	return
}

// call pkg-config and return the cflags and ldflags.
func pkgconfig(p *Package) ([]string, []string, error) {
	if len(p.CgoPkgConfig) == 0 {
		return nil, nil, nil // nothing to do
	}
	args := []string{
		"--cflags",
	}
	args = append(args, p.CgoPkgConfig...)
	var out bytes.Buffer
	err := runOut(&out, p.Dir, nil, "pkg-config", args...)
	if err != nil {
		return nil, nil, err
	}
	cflags := strings.Fields(out.String())

	args = []string{
		"--libs",
	}
	args = append(args, p.CgoPkgConfig...)
	out.Reset()
	err = runOut(&out, p.Dir, nil, "pkg-config", args...)
	if err != nil {
		return nil, nil, err
	}
	ldflags := strings.Fields(out.String())
	return cflags, ldflags, nil
}

func quoteFlags(flags []string) []string {
	quoted := make([]string, len(flags))
	for i, f := range flags {
		quoted[i] = strconv.Quote(f)
	}
	return quoted
}

// runcgo1 invokes the cgo tool to process pkg.CgoFiles.
func runcgo1(pkg *Package, cflags, ldflags []string) error {
	cgo := cgotool(pkg.Context)
	workdir := cgoworkdir(pkg)
	if err := mkdir(workdir); err != nil {
		return err
	}

	args := []string{"-objdir", workdir}
	switch {
	case gc14:
		args = append(args,
			"--",
			"-I", pkg.Dir,
		)
	case gc15:
		args = append(args,
			"-importpath", pkg.ImportPath,
			"--",
			"-I", workdir,
			"-I", pkg.Dir,
		)
	default:
		return fmt.Errorf("unsupported Go version: %v", runtime.Version)
	}
	args = append(args, cflags...)
	args = append(args, pkg.CgoFiles...)

	cgoenv := []string{
		"CGO_CFLAGS=" + strings.Join(quoteFlags(cflags), " "),
		"CGO_LDFLAGS=" + strings.Join(quoteFlags(ldflags), " "),
	}
	return run(pkg.Dir, cgoenv, cgo, args...)
}

// runcgo2 invokes the cgo tool to create _cgo_import.go
func runcgo2(pkg *Package, dynout, ofile string) error {
	cgo := cgotool(pkg.Context)
	workdir := cgoworkdir(pkg)

	args := []string{
		"-objdir", workdir,
	}
	switch {
	case gc14:
		args = append(args,
			"-dynimport", ofile,
			"-dynout", dynout,
		)
	case gc15:
		args = append(args,
			"-dynpackage", pkg.Name,
			"-dynimport", ofile,
			"-dynout", dynout,
		)
	default:
		return fmt.Errorf("unsuppored Go version: %v", runtime.Version)
	}
	return run(pkg.Dir, nil, cgo, args...)
}

// cgoworkdir returns the cgo working directory for this package.
func cgoworkdir(pkg *Package) string {
	return filepath.Join(Workdir(pkg), pkgname(pkg), "_cgo")
}

// gccCmd returns a gcc command line prefix.
func gccCmd(pkg *Package, objdir string) []string {
	return ccompilerCmd(pkg, "CC", defaultCC, objdir)
}

// gxxCmd returns a g++ command line prefix.
func gxxCmd(pkg *Package, objdir string) []string {
	return ccompilerCmd(pkg, "CXX", defaultCXX, objdir)
}

// ccompilerCmd returns a command line prefix for the given environment
// variable and using the default command when the variable is empty.
func ccompilerCmd(pkg *Package, envvar, defcmd, objdir string) []string {
	compiler := envList(envvar, defcmd)
	a := []string{compiler[0]}
	if objdir != "" {
		a = append(a, "-I", objdir)
	}
	a = append(a, compiler[1:]...)

	// Definitely want -fPIC but on Windows gcc complains
	// "-fPIC ignored for target (all code is position independent)"
	if pkg.gotargetos != "windows" {
		a = append(a, "-fPIC")
	}
	a = append(a, gccArchArgs(pkg.gotargetarch)...)
	// gcc-4.5 and beyond require explicit "-pthread" flag
	// for multithreading with pthread library.
	switch pkg.gotargetos {
	case "windows":
		a = append(a, "-mthreads")
	default:
		a = append(a, "-pthread")
	}

	if strings.Contains(a[0], "clang") {
		// disable ASCII art in clang errors, if possible
		a = append(a, "-fno-caret-diagnostics")
		// clang is too smart about command-line arguments
		a = append(a, "-Qunused-arguments")
	}

	// disable word wrapping in error messages
	a = append(a, "-fmessage-length=0")

	// On OS X, some of the compilers behave as if -fno-common
	// is always set, and the Mach-O linker in 6l/8l assumes this.
	// See https://golang.org/issue/3253.
	if pkg.gotargetos == "darwin" {
		a = append(a, "-fno-common")
	}

	return a
}

// gccArchArgs returns arguments to pass to gcc based on the architecture.
func gccArchArgs(goarch string) []string {
	switch goarch {
	case "386":
		return []string{"-m32"}
	case "amd64", "amd64p32":
		return []string{"-m64"}
	case "arm":
		return []string{"-marm"} // not thumb
	}
	return nil
}
