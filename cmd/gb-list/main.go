// gb-list lists the packages named by the import paths, one per line.
//
//     usage: gb list [-s] [-f format] [-json] [packages]
//
// The default output shows the package import path:
//
//     % gb list github.com/constabulary/...
//     github.com/constabulary/gb
//     github.com/constabulary/gb/cmd
//     github.com/constabulary/gb/cmd/gb
//     github.com/constabulary/gb/cmd/gb-env
//     github.com/constabulary/gb/cmd/gb-list
//
// The -f flag specifies an alternate format for the list, using the
// syntax of package template.  The default output is equivalent to -f
// '{{.ImportPath}}'. The struct being passed to the template is currently
// an instance of gb.Package. This structure is under active development
// and it's contents are not guarenteed to be stable.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/template"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
)

func main() {
	var (
		projectroot string
		format      string
		formatStdin bool
		jsonOutput  bool
	)
	flag.StringVar(&projectroot, "R", os.Getenv("GB_PROJECT_DIR"), "set the project root")
	flag.StringVar(&format, "f", "{{.ImportPath}}", "format template")
	flag.BoolVar(&formatStdin, "s", false, "read format from stdin")
	flag.BoolVar(&gb.Verbose, "v", gb.Verbose, "enable log levels below INFO level")
	flag.BoolVar(&jsonOutput, "json", false, "outputs json. WARNING: gb.Package structure is not stable and will change in future")

	flag.Parse()

	if formatStdin {
		var formatBuffer bytes.Buffer
		io.Copy(&formatBuffer, os.Stdin)
		format = formatBuffer.String()
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

	args := cmd.ImportPaths(ctx, cmd.MustGetwd(), flag.Args())
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
			gb.Fatalf("Error occurred during json encoding: %v", err)
		}
	} else {
		tmpl, err := template.New("list").Parse(format)
		if err != nil {
			gb.Fatalf("unable to parse template %q: %v", format, err)
		}

		for _, pkg := range pkgs {
			if err := tmpl.Execute(os.Stdout, pkg); err != nil {
				gb.Fatalf("unable to execute template: %v", err)
			}
			fmt.Fprintln(os.Stdout)
		}
	}
}
