package match

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/constabulary/gb/internal/debug"
)

// matchPattern(pattern)(name) reports whether
// name matches pattern.  Pattern is a limited glob
// pattern in which '...' means 'any string' and there
// is no other special syntax.
func matchPattern(pattern string) func(name string) bool {
	re := regexp.QuoteMeta(pattern)
	re = strings.Replace(re, `\.\.\.`, `.*`, -1)
	// Special case: foo/... matches foo too.
	if strings.HasSuffix(re, `/.*`) {
		re = re[:len(re)-len(`/.*`)] + `(/.*)?`
	}
	reg := regexp.MustCompile(`^` + re + `$`)
	return func(name string) bool {
		return reg.MatchString(name)
	}
}

// hasPathPrefix reports whether the path s begins with the
// elements in prefix.
func hasPathPrefix(s, prefix string) bool {
	switch {
	default:
		return false
	case len(s) == len(prefix):
		return s == prefix
	case len(s) > len(prefix):
		if prefix != "" && prefix[len(prefix)-1] == '/' {
			return strings.HasPrefix(s, prefix)
		}
		return s[len(prefix)] == '/' && s[:len(prefix)] == prefix
	}
}

// treeCanMatchPattern(pattern)(name) reports whether
// name or children of name can possibly match pattern.
// Pattern is the same limited glob accepted by matchPattern.
func treeCanMatchPattern(pattern string) func(name string) bool {
	wildCard := false
	if i := strings.Index(pattern, "..."); i >= 0 {
		wildCard = true
		pattern = pattern[:i]
	}
	return func(name string) bool {
		return len(name) <= len(pattern) && hasPathPrefix(pattern, name) ||
			wildCard && strings.HasPrefix(name, pattern)
	}
}

// matchPackages returns all the packages that can be found under the srcdir directory.
// The pattern is a path including "...".
func matchPackages(srcdir, pattern string) ([]string, error) {
	debug.Debugf("matchPackages: %v", pattern)
	match := matchPattern(pattern)
	treeCanMatch := treeCanMatchPattern(pattern)

	var pkgs []string

	src := srcdir + string(filepath.Separator)
	err := filepath.Walk(src, func(path string, fi os.FileInfo, err error) error {
		if err != nil || !fi.IsDir() || path == src {
			return nil
		}

		// Avoid .foo, _foo, and testdata directory trees.
		elem := fi.Name()
		if strings.HasPrefix(elem, ".") || strings.HasPrefix(elem, "_") || elem == "testdata" {
			return filepath.SkipDir
		}

		name := filepath.ToSlash(path[len(src):])
		if pattern == "std" && strings.Contains(name, ".") {
			return filepath.SkipDir
		}
		if !treeCanMatch(name) {
			return filepath.SkipDir
		}
		if match(name) {
			pkgs = append(pkgs, name)
		}
		return nil
	})
	return pkgs, err
}
