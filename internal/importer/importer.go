package importer

import (
	"bytes"
	"fmt"
	"io"
	"os"
	pathpkg "path"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"
)

type Context struct {
	GOOS       string // target architecture
	GOARCH     string // target operating system
	CgoEnabled bool   // whether cgo can be used

	// The build and release tags specify build constraints
	// that should be considered satisfied when processing +build lines.
	// Clients creating a new context may customize BuildTags, which
	// defaults to empty, but it is usually an error to customize ReleaseTags,
	// which defaults to the list of Go releases the current release is compatible with.
	// In addition to the BuildTags and ReleaseTags, build constraints
	// consider the values of GOARCH and GOOS as satisfied tags.
	BuildTags   []string
	ReleaseTags []string
}

type Importer struct {
	*Context
	Root string // root directory
}

func (i *Importer) Import(path string) (*Package, error) {
	if path == "" {
		return nil, fmt.Errorf("import %q: invalid import path", path)
	}

	if path == "." || path == ".." || strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") {
		return nil, fmt.Errorf("import %q: relative import not supported", path)
	}

	if strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("import %q: cannot import absolute path", path)
	}

	p := &Package{
		importer:   i,
		ImportPath: path,
		Standard:   i.Root == runtime.GOROOT(),
	}
	// if this is the stdlib, then search vendor first.
	// this isn't real vendor support, just enough to make net/http compile.
	if p.Standard {
		path := pathpkg.Join("vendor", path)
		dir := filepath.Join(i.Root, "src", filepath.FromSlash(path))
		fi, err := os.Stat(dir)
		if err == nil && fi.IsDir() {
			p.Dir = dir
			p.Root = i.Root
			p.ImportPath = path
			p.SrcRoot = filepath.Join(p.Root, "src")
			err = loadPackage(p)
			return p, err
		}
	}

	dir := filepath.Join(i.Root, "src", filepath.FromSlash(path))
	fi, err := os.Stat(dir)
	if err == nil {
		if fi.IsDir() {
			p.Dir = dir
			p.Root = i.Root
			p.SrcRoot = filepath.Join(p.Root, "src")
			err = loadPackage(p)
			return p, err
		}
		err = fmt.Errorf("import %q: not a directory", path)
	}
	return nil, err
}

// matchFile determines whether the file with the given name in the given directory
// should be included in the package being constructed.
// It returns the data read from the file.
// If allTags is non-nil, matchFile records any encountered build tag
// by setting allTags[tag] = true.
func (i *Importer) matchFile(path string, allTags map[string]bool) (match bool, data []byte, err error) {
	name := filepath.Base(path)
	if name[0] == '_' || name[0] == '.' {
		return
	}

	if !goodOSArchFile(i.GOOS, i.GOARCH, name, allTags) {
		return
	}

	read := func(path string, fn func(r io.Reader) ([]byte, error)) ([]byte, error) {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		data, err := fn(f)
		if err != nil {
			err = fmt.Errorf("read %s: %v", path, err)
		}
		return data, err
	}

	switch filepath.Ext(name) {
	case ".go":
		data, err = read(path, readImports)
		if err != nil {
			return
		}
		// Look for +build comments to accept or reject the file.
		if !i.shouldBuild(data, allTags) {
			return
		}

		match = true
		return

	case ".c", ".cc", ".cxx", ".cpp", ".m", ".s", ".h", ".hh", ".hpp", ".hxx", ".S", ".swig", ".swigcxx":
		// tentatively okay - read to make sure
		data, err = read(path, readComments)
		if err != nil {
			return
		}
		// Look for +build comments to accept or reject the file.
		if !i.shouldBuild(data, allTags) {
			return
		}

		match = true
		return

	case ".syso":
		// binary, no reading
		match = true
		return
	default:
		// skip
		return
	}
}

// shouldBuild reports whether it is okay to use this file,
// The rule is that in the file's leading run of // comments
// and blank lines, which must be followed by a blank line
// (to avoid including a Go package clause doc comment),
// lines beginning with '// +build' are taken as build directives.
//
// The file is accepted only if each such line lists something
// matching the file.  For example:
//
//      // +build windows linux
//
// marks the file as applicable only on Windows and Linux.
//
func (i *Importer) shouldBuild(content []byte, allTags map[string]bool) bool {
	// Pass 1. Identify leading run of // comments and blank lines,
	// which must be followed by a blank line.
	end := 0
	p := content
	for len(p) > 0 {
		line := p
		if i := bytes.IndexByte(line, '\n'); i >= 0 {
			line, p = line[:i], p[i+1:]
		} else {
			p = p[len(p):]
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 { // Blank line
			end = len(content) - len(p)
			continue
		}
		if !bytes.HasPrefix(line, []byte{'/', '/'}) { // Not comment line
			break
		}
	}
	content = content[:end]

	// Pass 2.  Process each line in the run.
	p = content
	allok := true
	for len(p) > 0 {
		line := p
		if i := bytes.IndexByte(line, '\n'); i >= 0 {
			line, p = line[:i], p[i+1:]
		} else {
			p = p[len(p):]
		}
		line = bytes.TrimSpace(line)
		if bytes.HasPrefix(line, []byte{'/', '/'}) {
			line = bytes.TrimSpace(line[2:])
			if len(line) > 0 && line[0] == '+' {
				// Looks like a comment +line.
				f := strings.Fields(string(line))
				if f[0] == "+build" {
					ok := false
					for _, tok := range f[1:] {
						if i.match(tok, allTags) {
							ok = true
						}
					}
					if !ok {
						allok = false
					}
				}
			}
		}
	}

	return allok
}

// match reports whether the name is one of:
//
//      $GOOS
//      $GOARCH
//      cgo (if cgo is enabled)
//      !cgo (if cgo is disabled)
//      ctxt.Compiler
//      !ctxt.Compiler
//      tag (if tag is listed in ctxt.BuildTags or ctxt.ReleaseTags)
//      !tag (if tag is not listed in ctxt.BuildTags or ctxt.ReleaseTags)
//      a comma-separated list of any of these
//
func (i *Importer) match(name string, allTags map[string]bool) bool {
	if name == "" {
		if allTags != nil {
			allTags[name] = true
		}
		return false
	}
	if n := strings.Index(name, ","); n >= 0 {
		// comma-separated list
		ok1 := i.match(name[:n], allTags)
		ok2 := i.match(name[n+1:], allTags)
		return ok1 && ok2
	}
	if strings.HasPrefix(name, "!!") { // bad syntax, reject always
		return false
	}
	if strings.HasPrefix(name, "!") { // negation
		return len(name) > 1 && !i.match(name[1:], allTags)
	}

	if allTags != nil {
		allTags[name] = true
	}

	// Tags must be letters, digits, underscores or dots.
	// Unlike in Go identifiers, all digits are fine (e.g., "386").
	for _, c := range name {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '_' && c != '.' {
			return false
		}
	}

	// special tags
	if i.CgoEnabled && name == "cgo" {
		return true
	}
	if name == i.GOOS || name == i.GOARCH || name == runtime.Compiler {
		return true
	}
	if i.GOOS == "android" && name == "linux" {
		return true
	}

	// other tags
	for _, tag := range i.BuildTags {
		if tag == name {
			return true
		}
	}
	for _, tag := range i.ReleaseTags {
		if tag == name {
			return true
		}
	}

	return false
}
