package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
	"github.com/pkg/errors"
)

var (
	format      string
	formatStdin bool
	jsonOutput  bool
)

func init() {
	registerCommand(&cmd.Command{
		Name:      "list",
		UsageLine: `list [-s] [-f format] [-json] [packages]`,
		Short:     "list the packages named by the importpaths",
		Long: `
List lists packages imported by the project.

The default output shows the package import paths:

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
		This structure is under active development and it's contents are not
		guaranteed to be stable.
	-s
		read format template from STDIN.
	-json
		prints output in structured JSON format. WARNING: gb.Package
		structure is not stable and will change in the future!
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
	if formatStdin {
		var formatBuffer bytes.Buffer
		io.Copy(&formatBuffer, os.Stdin)
		format = formatBuffer.String()
	}
	pkgs, err := resolveRootPackages(ctx, args...)
	if err != nil {
		log.Fatalf("unable to resolve: %v", err)
	}

	if jsonOutput {
		views := make([]*PackageView, 0, len(pkgs))
		for _, pkg := range pkgs {
			views = append(views, NewPackageView(pkg))
		}
		encoder := json.NewEncoder(os.Stdout)
		if err := encoder.Encode(views); err != nil {
			return errors.Wrap(err, "json encoding failed")
		}
	} else {
		fm := template.FuncMap{
			"join": strings.Join,
		}
		tmpl, err := template.New("list").Funcs(fm).Parse(format)
		if err != nil {
			return errors.Wrapf(err, "unable to parse template %q", format)
		}

		for _, pkg := range pkgs {
			if err := tmpl.Execute(os.Stdout, pkg); err != nil {
				return errors.Wrap(err, "unable to execute template")
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
