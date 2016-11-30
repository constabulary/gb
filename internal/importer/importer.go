package importer

import (
	"bytes"
	"fmt"
	"go/build"
	"os"
	pathpkg "path"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"

	"github.com/pkg/errors"
)

type Importer struct {
	*build.Context
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
		Standard: i.Root == runtime.GOROOT(),
	}

	loadPackage := func(importpath, dir string) error {
		pkg, err := i.Context.ImportDir(dir, 0)
		if err != nil {
			return err
		}
		p.Package = pkg
		p.ImportPath = importpath
		return nil
	}

	// if this is the stdlib, then search vendor first.
	// this isn't real vendor support, just enough to make net/http compile.
	if p.Standard {
		path := pathpkg.Join("vendor", path)
		dir := filepath.Join(i.Root, "src", filepath.FromSlash(path))
		fi, err := os.Stat(dir)
		if err == nil && fi.IsDir() {
			err := loadPackage(path, dir)
			return p, err
		}
	}

	dir := filepath.Join(i.Root, "src", filepath.FromSlash(path))
	fi, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, errors.Errorf("import %q: not a directory", path)
	}
	err = loadPackage(path, dir)
	return p, err
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
