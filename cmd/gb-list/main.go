package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"go/build"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
)

func main() {
	var (
		projectroot string
		format      string
	)
	flag.StringVar(&projectroot, "R", os.Getenv("GB_PROJECT_ROOT"), "set the project root")
	flag.StringVar(&format, "f", "{{.ImportPath}}\n", "format template")

	flag.Parse()

	tmpl, err := template.New("list").Parse(format)
	if err != nil {
		gb.Fatalf("unable to parse template %q: %v", format, err)
	}

	gopath := filepath.SplitList(os.Getenv("GOPATH"))
	root, err := cmd.FindProjectroot(projectroot, gopath)
	if err != nil {
		gb.Fatalf("could not locate project root: %v", err)
	}
	project := gb.NewProject(root)

	ctx, err := project.NewContext(
		gb.GcToolchain(),
	)
	if err != nil {
		gb.Fatalf("unable to construct context: %v", err)
	}

	args := cmd.ImportPaths(ctx, projectroot, flag.Args())
	pkgs, err := resolvePackages(ctx, projectroot, args...)
	if err != nil {
		gb.Fatalf("unable to resolve: %v", err)
	}
	for _, pkg := range pkgs {
		if err := tmpl.Execute(os.Stdout, pkg); err != nil {
			gb.Fatalf("unable to execute template: %v", err)
		}
	}
}

func resolvePackages(ctx *gb.Context, projectroot string, args ...string) ([]*gb.Package, error) {
	var pkgs []*gb.Package
	for _, arg := range args {
		if arg == "." {
			var err error
			arg, err = filepath.Rel(ctx.Srcdirs()[0], projectroot)
			if err != nil {
				return pkgs, err
			}
		}
		pkg, err := ctx.ResolvePackage(arg)
		if err != nil {
			if _, ok := err.(*build.NoGoError); ok {
				gb.Debugf("skipping %q", arg)
				continue
			}
			return pkgs, fmt.Errorf("failed to resolve package %q: %v", arg, err)
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs, nil
}
