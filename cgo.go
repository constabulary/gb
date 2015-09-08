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

	defun := filepath.Join(pkg.Objdir(), "_cgo_defun.o")
	rundefun := Action{
		Name: "cc: " + pkg.ImportPath + ": _cgo_defun_c",
		Deps: runcgo1,
		Task: TaskFn(func() error {
			return pkg.tc.Cc(pkg, defun, filepath.Join(pkg.Objdir(), "_cgo_defun.c"))
		}),
	}

	cgofiles := []string{filepath.Join(pkg.Objdir(), "_cgo_gotypes.go")}
	for _, f := range pkg.CgoFiles {
		cgofiles = append(cgofiles, filepath.Join(pkg.Objdir(), stripext(f)+".cgo1.go"))
	}
	cfiles := []string{
		filepath.Join(pkg.Objdir(), "_cgo_main.c"),
		filepath.Join(pkg.Objdir(), "_cgo_export.c"),
	}
	cfiles = append(cfiles, pkg.CFiles...)

	for _, f := range pkg.CgoFiles {
		cfiles = append(cfiles, filepath.Join(pkg.Objdir(), stripext(f)+".cgo2.c"))
	}

	cflags := append(cgoCPPFLAGS, cgoCFLAGS...)
	cxxflags := append(cgoCPPFLAGS, cgoCXXFLAGS...)
	gcc1, ofiles := cgocc(pkg, cflags, cxxflags, cfiles, pkg.CXXFiles, runcgo1...)
	ofile := filepath.Join(filepath.Dir(ofiles[0]), "_cgo_.o")
	gcc2 := Action{
		Name: "rungcc2: " + pkg.ImportPath + ": _cgo_.o",
		Deps: gcc1,
		Task: TaskFn(func() error {
			return rungcc2(pkg, cgoCFLAGS, cgoLDFLAGS, ofile, ofiles)
		}),
	}

	dynout := filepath.Join(pkg.Objdir(), "_cgo_import.c")
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
			return rungcc3(pkg.Context, pkg.Dir, allo, ofiles[1:]) // skip _cgo_main.o
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

	cgofiles := []string{filepath.Join(pkg.Objdir(), "_cgo_gotypes.go")}
	for _, f := range pkg.CgoFiles {
		cgofiles = append(cgofiles, filepath.Join(pkg.Objdir(), stripext(f)+".cgo1.go"))
	}
	cfiles := []string{
		filepath.Join(pkg.Objdir(), "_cgo_main.c"),
		filepath.Join(pkg.Objdir(), "_cgo_export.c"),
	}
	cfiles = append(cfiles, pkg.CFiles...)

	for _, f := range pkg.CgoFiles {
		cfiles = append(cfiles, filepath.Join(pkg.Objdir(), stripext(f)+".cgo2.c"))
	}

	cflags := append(cgoCPPFLAGS, cgoCFLAGS...)
	cxxflags := append(cgoCPPFLAGS, cgoCXXFLAGS...)
	gcc1, ofiles := cgocc(pkg, cflags, cxxflags, cfiles, pkg.CXXFiles, runcgo1...)
	ofile := filepath.Join(filepath.Dir(ofiles[0]), "_cgo_.o")
	gcc2 := Action{
		Name: "rungcc2: " + pkg.ImportPath + ": _cgo_.o",
		Deps: gcc1,
		Task: TaskFn(func() error {
			return rungcc2(pkg, cgoCFLAGS, cgoLDFLAGS, ofile, ofiles)
		}),
	}

	dynout := filepath.Join(pkg.Objdir(), "_cgo_import.go")
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
			return rungcc3(pkg.Context, pkg.Dir, allo, ofiles[1:]) // skip _cgo_main.o
		}),
	}

	return &action, []string{allo}, cgofiles, nil
}

// cgocc compiles all .c files.
// TODO(dfc) cxx not done
func cgocc(pkg *Package, cflags, cxxflags, cfiles, cxxfiles []string, deps ...*Action) ([]*Action, []string) {
	var cc []*Action
	var ofiles []string
	for _, cfile := range cfiles {
		cfile := cfile
		ofile := filepath.Join(pkg.Objdir(), stripext(filepath.Base(cfile))+".o")
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
		ofile := filepath.Join(pkg.Objdir(), stripext(filepath.Base(cxxfile))+".o")
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
	args := []string{"-g", "-O2", "-fPIC", "-pthread", "-fmessage-length=0",
		"-I", pkg.Dir,
		"-I", filepath.Dir(ofile),
	}
	args = append(args, cgoCFLAGS...)
	args = append(args,
		"-o", ofile,
		"-c", cfile,
	)
	t0 := time.Now()
	err := run(pkg.Dir, nil, gcc(), args...)
	pkg.Record("gcc1", time.Since(t0))
	return err
}

// rungpp1 invokes g++ to compile cfile into ofile
func rungpp1(pkg *Package, cgoCFLAGS []string, ofile, cfile string) error {
	args := []string{"-g", "-O2", "-fPIC", "-pthread", "-fmessage-length=0",
		"-I", pkg.Dir,
		"-I", filepath.Dir(ofile),
	}
	args = append(args, cgoCFLAGS...)
	args = append(args,
		"-o", ofile,
		"-c", cfile,
	)
	t0 := time.Now()
	err := run(pkg.Dir, nil, "g++", args...) // TODO(dfc) hack
	pkg.Record("gcc1", time.Since(t0))
	return err
}

// rungcc2 links the o files from rungcc1 into a single _cgo_.o.
func rungcc2(pkg *Package, cgoCFLAGS, cgoLDFLAGS []string, ofile string, ofiles []string) error {
	args := []string{
		"-fPIC", "-fmessage-length=0",
	}
	if !isClang() {
		args = append(args, "-pthread")
	}
	args = append(args, "-o", ofile)
	args = append(args, ofiles...)
	args = append(args, cgoLDFLAGS...) // this has to go at the end, because reasons!
	t0 := time.Now()
	err := run(pkg.Dir, nil, gcc(), args...)
	pkg.Record("gcc2", time.Since(t0))
	return err
}

// rungcc3 links all previous ofiles together with libgcc into a single _all.o.
func rungcc3(ctx *Context, dir string, ofile string, ofiles []string) error {
	args := []string{
		"-fPIC", "-fmessage-length=0",
	}
	if !isClang() {
		args = append(args, "-pthread")
	}
	args = append(args, "-g", "-O2", "-o", ofile)
	args = append(args, ofiles...)
	args = append(args, "-Wl,-r", "-nostdlib")
	if !isClang() {
		libgcc, err := libgcc(ctx)
		if err != nil {
			return nil
		}
		args = append(args, libgcc)
	} else {
		// explicitly disable build-id when using clang
		args = append(args, "-Wl,--build-id=none")
	}
	t0 := time.Now()
	err := run(dir, nil, gcc(), args...)
	ctx.Record("gcc3", time.Since(t0))
	return err
}

// libgcc returns the value of gcc -print-libgcc-file-name.
func libgcc(ctx *Context) (string, error) {
	args := []string{
		"-print-libgcc-file-name",
	}
	var buf bytes.Buffer
	err := runOut(&buf, ".", nil, gcc(), args...)
	return strings.TrimSpace(buf.String()), err
}

func cgotool(ctx *Context) string {
	// TODO(dfc) need ctx.GOROOT method
	return filepath.Join(ctx.Context.GOROOT, "pkg", "tool", ctx.gohostos+"_"+ctx.gohostarch, "cgo")
}

func gcc() string {
	return gccBaseCmd()[0] // TODO(dfc) handle gcc wrappers properly
}

func isClang() bool {
	return strings.HasPrefix(gcc(), "clang")
}

// gccBaseCmd returns the start of the compiler command line.
// It uses $CC if set, or else $GCC, or else the default
// compiler for the operating system is used.
func gccBaseCmd() []string {
	// Use $CC if set, since that's what the build uses.
	if ret := strings.Fields(os.Getenv("CC")); len(ret) > 0 {
		return ret
	}
	// Try $GCC if set, since that's what we used to use.
	if ret := strings.Fields(os.Getenv("GCC")); len(ret) > 0 {
		return ret
	}
	return strings.Fields(defaultCC)
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
	objdir := pkg.Objdir()
	if err := mkdir(objdir); err != nil {
		return err
	}

	args := []string{"-objdir", objdir}
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
			"-I", objdir,
			"-I", pkg.Dir,
		)
	default:
		return fmt.Errorf("unsuppored Go version: %v", runtime.Version)
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
	objdir := pkg.Objdir()

	args := []string{
		"-objdir", objdir,
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
