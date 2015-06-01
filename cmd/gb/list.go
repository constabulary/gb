package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"text/template"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
)

var (
	projectroot string
	format      string
	formatStdin bool
	jsonOutput  bool
)

func init() {
	registerCommand(&cmd.Command{
		Name:      "list",
		UsageLine: `list [-s] [-f format] [-json] [packages]`,
		Short:     "list the packages named by the importpaths",
		Long: `list lists packages.

The default output shows the package import path:

	% gb list github.com/constabulary/...
	github.com/constabulary/gb
	github.com/constabulary/gb/cmd
	github.com/constabulary/gb/cmd/gb
	github.com/constabulary/gb/cmd/gb-env
	github.com/constabulary/gb/cmd/gb-list

Flags:
	-f
		alternate format for the list, using the syntax of package template.
		The default output is equivalent to -f '{{.ImportPath}}'. The struct
		being passed to the template is currently an instance of gb.Package.
		This structure is under active development and it'As contents are not
		guarenteed to be stable.
`,
		Run: list,
		AddFlags: func(fs *flag.FlagSet) {
			fs.StringVar(&format, "f", "{{.ImportPath}}", "format template")
			fs.BoolVar(&formatStdin, "s", false, "read format from stdin")
			fs.BoolVar(&jsonOutput, "json", false, "outputs json. WARNING: gb.Package structure is not stable and will change in future")
		},
	})
}

func list(ctx *gb.Context, args []string) error {
	gb.Debugf("list: %v", args)
	if formatStdin {
		var formatBuffer bytes.Buffer
		io.Copy(&formatBuffer, os.Stdin)
		format = formatBuffer.String()
	}
	args = cmd.ImportPaths(ctx, cmd.MustGetwd(), args)
	pkgs, err := cmd.ResolvePackages(ctx, args...)
	if err != nil {
		gb.Fatalf("unable to resolve: %v", err)
	}

	if jsonOutput {
		views := make([]*PackageView, 0, len(pkgs))
		for _, pkg := range pkgs {
			views = append(views, NewPackageView(pkg))
		}
		encoder := json.NewEncoder(os.Stdout)
		if err := encoder.Encode(views); err != nil {
			return fmt.Errorf("Error occurred during json encoding: %v", err)
		}
	} else {
		tmpl, err := template.New("list").Parse(format)
		if err != nil {
			return fmt.Errorf("unable to parse template %q: %v", format, err)
		}

		for _, pkg := range pkgs {
			if err := tmpl.Execute(os.Stdout, pkg); err != nil {
				return fmt.Errorf("unable to execute template: %v", err)
			}
			fmt.Fprintln(os.Stdout)
		}
	}
	return nil
}

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
