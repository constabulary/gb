package gb

import (
	"fmt"
	"strings"
)

type loader struct {
	*Context
	importer

	pkgs map[string]*Package // map of package paths to resolved packages
}

// loadPackage recursively resolves path as a package. If successful loadPackage
// records the package in the Context's internal package cache.
func (l *loader) loadPackage(stack []string, path string) (*Package, error) {
	// sanity check
	if path == "." || path == ".." || strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") {
		return nil, fmt.Errorf("%q is not a valid import path", path)
	}
	if pkg, ok := l.pkgs[path]; ok {
		// already loaded, just return
		return pkg, nil
	}

	push := func(path string) {
		stack = append(stack, path)
	}
	pop := func(path string) {
		stack = stack[:len(stack)-1]
	}
	onStack := func(path string) bool {
		for _, p := range stack {
			if p == path {
				return true
			}
		}
		return false
	}

	p, err := l.Import(path)
	if err != nil {
		return nil, err
	}

	standard := p.Goroot && p.ImportPath != "" && !strings.HasPrefix(p.ImportPath, ".") // TODO(dfc) ensure relative imports never get this far
	push(p.ImportPath)
	var stale bool
	for i, im := range p.Imports {
		if onStack(im) {
			push(im)
			return nil, fmt.Errorf("import cycle detected: %s", strings.Join(stack, " -> "))
		}
		pkg, err := l.loadPackage(stack, im)
		if err != nil {
			return nil, err
		}

		// update the import path as the import may have been discovered via vendoring.
		p.Imports[i] = pkg.ImportPath
		stale = stale || pkg.Stale
	}
	pop(p.ImportPath)

	pkg := Package{
		Context:  l.Context,
		Package:  p,
		Standard: standard,
	}
	pkg.Stale = stale || isStale(&pkg)
	l.pkgs[p.ImportPath] = &pkg
	return &pkg, nil
}
