package main

import (
	"flag"
	"fmt"
	"html/template"
	"os"
	"text/tabwriter"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
	"github.com/constabulary/gb/internal/vendor"
	"github.com/pkg/errors"
)

var format string

var cmdList = &cmd.Command{
	Name:      "list",
	UsageLine: "list [-f format]",
	Short:     "lists dependencies, one per line",
	Long: `gb vendor list formats lists the contents of the manifest file.

The output

Flags:
	-f
		controls the template used for printing each manifest entry. If not supplied
		the default value is "{{.Importpath}}\t{{.Repository}}{{.Path}}\t{{.Branch}}\t{{.Revision}}"

`,
	Run: func(ctx *gb.Context, args []string) error {
		m, err := vendor.ReadManifest(manifestFile(ctx))
		if err != nil {
			return errors.Wrap(err, "could not load manifest")
		}
		tmpl, err := template.New("list").Parse(format)
		if err != nil {
			return errors.Wrapf(err, "unable to parse template %q", format)
		}
		w := tabwriter.NewWriter(os.Stdout, 1, 2, 1, ' ', 0)
		for _, dep := range m.Dependencies {
			if err := tmpl.Execute(w, dep); err != nil {
				return errors.Wrap(err, "unable to execute template")
			}
			fmt.Fprintln(w)
		}
		return w.Flush()
	},
	AddFlags: func(fs *flag.FlagSet) {
		fs.StringVar(&format, "f", "{{.Importpath}}\t{{.Repository}}{{.Path}}\t{{.Branch}}\t{{.Revision}}", "format template")
	},
}
