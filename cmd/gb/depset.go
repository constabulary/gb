package main

import (
	"fmt"
	"go/build"
	"path/filepath"
	"runtime"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
	"github.com/constabulary/gb/vendor"
)

func init() {
	registerCommand(&cmd.Command{
		Name: "depset",
		Run:  depset,
	})
}

func depset(ctx *gb.Context, args []string) error {
	paths := []struct {
		Root, Prefix string
	}{
		{filepath.Join(runtime.GOROOT(), "src"), ""},
		{filepath.Join(ctx.Projectdir(), "src"), ""},
	}
	m, err := vendor.ReadManifest(filepath.Join("vendor", "manifest"))
	if err != nil {
		return err
	}
	for _, d := range m.Dependencies {
		paths = append(paths, struct{ Root, Prefix string }{filepath.Join(ctx.Projectdir(), "vendor", "src", filepath.FromSlash(d.Importpath)), filepath.FromSlash(d.Importpath)})
	}

	dsm, err := vendor.LoadPaths(paths...)
	if err != nil {
		return err
	}
	for _, set := range dsm {
		fmt.Printf("%s (%s)\n", set.Root, set.Prefix)
		for _, p := range set.Pkgs {
			fmt.Printf("\t%s (%s)\n", p.ImportPath, p.Name)
			fmt.Printf("\t\timports: %s\n", p.Imports)
		}
	}

	root := paths[1] // $PROJECT/src
	rs := dsm[root.Root].Pkgs

	fmt.Println("missing:")
	for missing := range findMissing(pkgs(rs), dsm) {
		fmt.Printf("\t%s\n", missing)
	}

	fmt.Println("orphaned:")
	for orphan := range findOrphaned(pkgs(rs), dsm) {
		fmt.Printf("\t%s\n", orphan)
	}

	return nil
}

func keys(m map[string]bool) []string {
	var s []string
	for k := range m {
		s = append(s, k)
	}
	return s
}

func pkgs(m map[string]*vendor.Pkg) []*vendor.Pkg {
	var p []*vendor.Pkg
	for _, v := range m {
		p = append(p, v)
	}
	return p
}

func findMissing(pkgs []*vendor.Pkg, dsm map[string]*vendor.Depset) map[string]bool {
	missing := make(map[string]bool)
	imports := make(map[string]*vendor.Pkg)
	for _, s := range dsm {
		for _, p := range s.Pkgs {
			imports[p.ImportPath] = p
		}
	}

	// make fake C package for cgo
	imports["C"] = &vendor.Pkg{
		Depset: nil, // probably a bad idea
		Package: &build.Package{
			Name: "C",
		},
	}
	stk := make(map[string]bool)
	push := func(v string) {
		if stk[v] {
			panic(fmt.Sprintln("import loop:", v, stk))
		}
		stk[v] = true
	}
	pop := func(v string) {
		if !stk[v] {
			panic(fmt.Sprintln("impossible pop:", v, stk))
		}
		delete(stk, v)
	}

	// checked records import paths who's dependencies are all present
	checked := make(map[string]bool)

	var fn func(string)
	fn = func(importpath string) {
		p, ok := imports[importpath]
		if !ok {
			missing[importpath] = true
			return
		}

		// have we already walked this arm, if so, skip it
		if checked[importpath] {
			return
		}

		sz := len(missing)
		push(importpath)
		for _, i := range p.Imports {
			if i == importpath {
				continue
			}
			fn(i)
		}

		// if the size of the missing map has not changed
		// this entire subtree is complete, mark it as such
		if len(missing) == sz {
			checked[importpath] = true
		}
		pop(importpath)
	}
	for _, pkg := range pkgs {
		fn(pkg.ImportPath)
	}
	return missing
}

func findOrphaned(pkgs []*vendor.Pkg, dsm map[string]*vendor.Depset) map[string]bool {
	missing := make(map[string]bool)
	imports := make(map[string]*vendor.Pkg)
	for _, s := range dsm {
		for _, p := range s.Pkgs {
			imports[p.ImportPath] = p
		}
	}

	orphans := make(map[string]bool)
	for k := range dsm {
		orphans[k] = true
	}

	// make fake C package for cgo
	imports["C"] = &vendor.Pkg{
		Depset: new(vendor.Depset),
		Package: &build.Package{
			Name: "C",
		},
	}
	stk := make(map[string]bool)
	push := func(v string) {
		if stk[v] {
			panic(fmt.Sprintln("import loop:", v, stk))
		}
		stk[v] = true
	}
	pop := func(v string) {
		if !stk[v] {
			panic(fmt.Sprintln("impossible pop:", v, stk))
		}
		delete(stk, v)
	}

	// checked records import paths who's dependencies are all present
	checked := make(map[string]bool)

	var fn func(string)
	fn = func(importpath string) {
		p, ok := imports[importpath]
		if !ok {
			missing[importpath] = true
			return
		}

		// delete the pkg's depset, as it is referenced
		delete(orphans, p.Depset.Root)

		// have we already walked this arm, if so, skip it
		if checked[importpath] {
			return
		}

		sz := len(missing)
		push(importpath)
		for _, i := range p.Imports {
			if i == importpath {
				continue
			}
			fn(i)
		}

		// if the size of the missing map has not changed
		// this entire subtree is complete, mark it as such
		if len(missing) == sz {
			checked[importpath] = true
		}
		pop(importpath)
	}
	for _, pkg := range pkgs {
		fn(pkg.ImportPath)
	}
	return orphans
}
