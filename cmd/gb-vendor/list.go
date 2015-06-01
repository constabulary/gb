package main

import (
	"flag"
	"fmt"
	"html/template"
	"os"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
	"github.com/constabulary/gb/cmd/gb-vendor/vendor"
)

func init() {
	registerCommand(List)
}

var format string

var List = &cmd.Command{
	Name:      "list",
	Short: "lists the packages named by the import paths, one per line.",
	Run: func(ctx *gb.Context, args []string) error {
		m, err := vendor.ReadManifest(manifestFile(ctx))
		if err != nil {
			return fmt.Errorf("could not load manifest: %v", err)
		}
		tmpl, err := template.New("list").Parse(format)
		if err != nil {
			return fmt.Errorf("unable to parse template %q: %v", format, err)
		}

		for _, dep := range m.Dependencies {
			if err := tmpl.Execute(os.Stdout, dep); err != nil {
				return fmt.Errorf("unable to execute template: %v", err)
			}
			fmt.Fprintln(os.Stdout)
		}
		return nil
	},
	AddFlags: func(fs *flag.FlagSet) {
		fs.StringVar(&format, "f", "{{.Importpath}}\t{{.Repository}}{{.Path}}\t{{.Branch}}\t{{.Revision}}", "format template")
	},
}
