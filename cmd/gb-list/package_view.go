package main

import "github.com/constabulary/gb"

// PackageView represents a package shown by list command in JSON format.
// It is not stable and may be subject to change.
type PackageView struct {
	Dir         string
	ImportPath  string
	Name        string
	Root        string
	GoFiles     []string
	Imports     []string
	TestGoFiles []string
	TestImports []string
}

// NewPackageView creates a *PackageView from gb Package.
func NewPackageView(pkg *gb.Package) *PackageView {
	return &PackageView{
		Dir:         pkg.Dir,
		ImportPath:  pkg.ImportPath,
		Name:        pkg.Name,
		Root:        pkg.Root,
		GoFiles:     pkg.GoFiles,
		Imports:     pkg.Package.Imports,
		TestGoFiles: pkg.TestGoFiles,
		TestImports: pkg.TestImports,
	}
}
