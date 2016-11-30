package importer

import (

	// for build.Default

	"path/filepath"
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

func (i *Importer) loadPackage(p *Package) error {
	pkg, err := i.Context.ImportDir(p.Dir, 0)
	if err != nil {
		return err
	}
	importpath, err := filepath.Rel(p.SrcRoot, p.Dir)
	if err != nil {
		return errors.WithStack(err)
	}
	pkg.ImportPath = filepath.ToSlash(importpath)
	p.Package = pkg
	return nil
}
