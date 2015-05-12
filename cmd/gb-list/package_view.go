package main

import "github.com/constabulary/gb"

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
