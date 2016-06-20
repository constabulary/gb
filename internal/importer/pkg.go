package importer

import (
	"bufio"
	"go/ast" // for build.Default
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

var knownOS = map[string]bool{
	"android":   true,
	"darwin":    true,
	"dragonfly": true,
	"freebsd":   true,
	"linux":     true,
	"nacl":      true,
	"netbsd":    true,
	"openbsd":   true,
	"plan9":     true,
	"solaris":   true,
	"windows":   true,
}

var knownArch = map[string]bool{
	"386":         true,
	"amd64":       true,
	"amd64p32":    true,
	"arm":         true,
	"armbe":       true,
	"arm64":       true,
	"arm64be":     true,
	"mips":        true,
	"mipsle":      true,
	"mips64":      true,
	"mips64le":    true,
	"mips64p32":   true,
	"mips64p32le": true,
	"ppc":         true,
	"ppc64":       true,
	"ppc64le":     true,
	"s390":        true,
	"s390x":       true,
	"sparc":       true,
	"sparc64":     true,
}

// goodOSArchFile returns false if the name contains a $GOOS or $GOARCH
// suffix which does not match the current system.
// The recognized name formats are:
//
//     name_$(GOOS).*
//     name_$(GOARCH).*
//     name_$(GOOS)_$(GOARCH).*
//     name_$(GOOS)_test.*
//     name_$(GOARCH)_test.*
//     name_$(GOOS)_$(GOARCH)_test.*
//
// An exception: if GOOS=android, then files with GOOS=linux are also matched.
func goodOSArchFile(goos, goarch, name string, allTags map[string]bool) bool {
	// Before Go 1.4, a file called "linux.go" would be equivalent to having a
	// build tag "linux" in that file. For Go 1.4 and beyond, we require this
	// auto-tagging to apply only to files with a non-empty prefix, so
	// "foo_linux.go" is tagged but "linux.go" is not. This allows new operating
	// systems, such as android, to arrive without breaking existing code with
	// innocuous source code in "android.go". The easiest fix: cut everything
	// in the name before the initial _.
	i := strings.Index(name, "_")
	if i < 0 {
		return true
	}
	name = name[i:] // ignore everything before first _

	// strip extension
	if dot := strings.Index(name, "."); dot != -1 {
		name = name[:dot]
	}

	l := strings.Split(name, "_")
	if n := len(l); n > 0 && l[n-1] == "test" {
		l = l[:n-1]
	}
	n := len(l)
	switch {
	case n >= 2 && knownOS[l[n-2]] && knownArch[l[n-1]]:
		if allTags != nil {
			allTags[l[n-2]] = true
			allTags[l[n-1]] = true
		}
		if l[n-1] != goarch {
			return false
		}
		if goos == "android" && l[n-2] == "linux" {
			return true
		}
		return l[n-2] == goos
	case n >= 1 && knownOS[l[n-1]]:
		if allTags != nil {
			allTags[l[n-1]] = true
		}
		if goos == "android" && l[n-1] == "linux" {
			return true
		}
		return l[n-1] == goos
	case n >= 1 && knownArch[l[n-1]]:
		if allTags != nil {
			allTags[l[n-1]] = true
		}
		return l[n-1] == goarch
	default:
		return true
	}
}

type byName []os.FileInfo

func (x byName) Len() int           { return len(x) }
func (x byName) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }
func (x byName) Less(i, j int) bool { return x[i].Name() < x[j].Name() }

func loadPackage(p *Package) error {
	dir, err := os.Open(p.Dir)
	if err != nil {
		return errors.Wrap(err, "unable open directory")
	}
	defer dir.Close()

	dents, err := dir.Readdir(-1)
	if err != nil {
		return errors.Wrap(err, "unable read directory")
	}

	var Sfiles []string // files with ".S" (capital S)
	var firstFile string
	imported := make(map[string][]token.Position)
	testImported := make(map[string][]token.Position)
	xTestImported := make(map[string][]token.Position)
	allTags := make(map[string]bool)
	fset := token.NewFileSet()

	// cmd/gb expects file names to be sorted ... this seems artificial
	sort.Sort(byName(dents))
	for _, fi := range dents {
		if fi.IsDir() {
			continue
		}

		name := fi.Name()
		path := filepath.Join(p.Dir, name)
		match, data, err := p.matchFile(path, allTags)
		if err != nil {
			return err
		}
		ext := filepath.Ext(name)
		if !match {
			if ext == ".go" {
				p.IgnoredGoFiles = append(p.IgnoredGoFiles, name)
			}
			continue
		}

		switch ext {
		case ".c":
			p.CFiles = append(p.CFiles, name)
		case ".cc", ".cpp", ".cxx":
			p.CXXFiles = append(p.CXXFiles, name)
		case ".m":
			p.MFiles = append(p.MFiles, name)
		case ".h", ".hh", ".hpp", ".hxx":
			p.HFiles = append(p.HFiles, name)
		case ".s":
			p.SFiles = append(p.SFiles, name)
		case ".S":
			Sfiles = append(Sfiles, name)
		case ".swig":
			p.SwigFiles = append(p.SwigFiles, name)
		case ".swigcxx":
			p.SwigCXXFiles = append(p.SwigCXXFiles, name)
		case ".syso":
			// binary objects to add to package archive
			// Likely of the form foo_windows.syso, but
			// the name was vetted above with goodOSArchFile.
			p.SysoFiles = append(p.SysoFiles, name)
		default:
			pf, err := parser.ParseFile(fset, path, data, parser.ImportsOnly|parser.ParseComments)
			if err != nil {
				return err
			}

			pkg := pf.Name.Name
			if pkg == "documentation" {
				p.IgnoredGoFiles = append(p.IgnoredGoFiles, name)
				continue
			}

			isTest := strings.HasSuffix(name, "_test.go")
			isXTest := false
			if isTest && strings.HasSuffix(pkg, "_test") {
				isXTest = true
				pkg = pkg[:len(pkg)-len("_test")]
			}

			if p.Name == "" {
				p.Name = pkg
				firstFile = name
			} else if pkg != p.Name {
				return &MultiplePackageError{
					Dir:      p.Dir,
					Packages: []string{p.Name, pkg},
					Files:    []string{firstFile, name},
				}
			}
			// Record imports and information about cgo.
			isCgo := false
			for _, decl := range pf.Decls {
				d, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}
				for _, dspec := range d.Specs {
					spec, ok := dspec.(*ast.ImportSpec)
					if !ok {
						continue
					}
					quoted := spec.Path.Value
					path, err := strconv.Unquote(quoted)
					if err != nil {
						return errors.Errorf("%q: invalid quoted string: %q", path, quoted)
					}
					if isXTest {
						xTestImported[path] = append(xTestImported[path], fset.Position(spec.Pos()))
					} else if isTest {
						testImported[path] = append(testImported[path], fset.Position(spec.Pos()))
					} else {
						imported[path] = append(imported[path], fset.Position(spec.Pos()))
					}
					if path == "C" {
						if isTest {
							return errors.Errorf("use of cgo in test %s not supported", path)
						}
						cg := spec.Doc
						if cg == nil && len(d.Specs) == 1 {
							cg = d.Doc
						}
						if cg != nil {
							if err := saveCgo(p, path, cg); err != nil {
								return err
							}
						}
						isCgo = true
					}
				}
			}
			switch {
			case isCgo:
				allTags["cgo"] = true
				if p.importer.(*Importer).CgoEnabled {
					p.CgoFiles = append(p.CgoFiles, name)
				} else {
					p.IgnoredGoFiles = append(p.IgnoredGoFiles, name)
				}
			case isXTest:
				p.XTestGoFiles = append(p.XTestGoFiles, name)
			case isTest:
				p.TestGoFiles = append(p.TestGoFiles, name)
			default:
				p.GoFiles = append(p.GoFiles, name)
			}
		}
	}
	if len(p.GoFiles)+len(p.CgoFiles)+len(p.TestGoFiles)+len(p.XTestGoFiles) == 0 {
		return &NoGoError{p.Dir}
	}

	for tag := range allTags {
		p.AllTags = append(p.AllTags, tag)
	}
	sort.Strings(p.AllTags)

	p.Imports, p.ImportPos = cleanImports(imported)
	p.TestImports, p.TestImportPos = cleanImports(testImported)
	p.XTestImports, p.XTestImportPos = cleanImports(xTestImported)

	// add the .S files only if we are using cgo
	// (which means gcc will compile them).
	// The standard assemblers expect .s files.
	if len(p.CgoFiles) > 0 {
		p.SFiles = append(p.SFiles, Sfiles...)
		sort.Strings(p.SFiles)
	}
	return nil
}

// saveCgo saves the information from the #cgo lines in the import "C" comment.
// These lines set CFLAGS, CPPFLAGS, CXXFLAGS and LDFLAGS and pkg-config directives
// that affect the way cgo's C code is built.
func saveCgo(di *Package, filename string, cg *ast.CommentGroup) error {
	r := strings.NewReader(cg.Text())
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()

		// Line is
		//	#cgo [GOOS/GOARCH...] LDFLAGS: stuff
		//
		line = strings.TrimSpace(line)
		if len(line) < 5 || line[:4] != "#cgo" || (line[4] != ' ' && line[4] != '\t') {
			continue
		}

		// Split at colon.
		line = strings.TrimSpace(line[4:])
		i := strings.Index(line, ":")
		if i < 0 {
			return errors.Errorf("%s: invalid #cgo line: %s", filename, sc.Text())
		}
		line, argstr := line[:i], line[i+1:]

		// Parse GOOS/GOARCH stuff.
		f := strings.Fields(line)
		if len(f) < 1 {
			return errors.Errorf("%s: invalid #cgo line: %s", filename, sc.Text())
		}

		cond, verb := f[:len(f)-1], f[len(f)-1]
		if len(cond) > 0 {
			ok := false
			for _, c := range cond {
				if di.match(c, nil) {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}

		args, err := splitQuoted(argstr)
		if err != nil {
			return errors.Wrapf(err, "%s: invalid #cgo line: %s", filename, sc.Text())
		}
		for i, arg := range args {
			arg, ok := expandSrcDir(arg, di.Dir)
			if !ok {
				return errors.Errorf("%s: malformed #cgo argument: %s", filename, arg)
			}
			args[i] = arg
		}

		switch verb {
		case "CFLAGS":
			di.CgoCFLAGS = append(di.CgoCFLAGS, args...)
		case "CPPFLAGS":
			di.CgoCPPFLAGS = append(di.CgoCPPFLAGS, args...)
		case "CXXFLAGS":
			di.CgoCXXFLAGS = append(di.CgoCXXFLAGS, args...)
		case "LDFLAGS":
			di.CgoLDFLAGS = append(di.CgoLDFLAGS, args...)
		case "pkg-config":
			di.CgoPkgConfig = append(di.CgoPkgConfig, args...)
		default:
			return errors.Errorf("%s: invalid #cgo verb: %s", filename, sc.Text())
		}
	}
	return sc.Err()
}

func cleanImports(m map[string][]token.Position) ([]string, map[string][]token.Position) {
	if len(m) == 0 {
		return nil, nil
	}
	all := make([]string, 0, len(m))
	for path := range m {
		all = append(all, path)
	}
	sort.Strings(all)
	return all, m
}
