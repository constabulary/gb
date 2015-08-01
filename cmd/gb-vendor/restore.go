package main

import (
	"flag"
	"fmt"
	"path/filepath"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
	"github.com/constabulary/gb/vendor"
)

func addRestoreFlags(fs *flag.FlagSet) {
	fs.BoolVar(&insecure, "precaire", false, "allow the use of insecure protocols")
}

var cmdRestore = &cmd.Command{
	Name:      "restore",
	UsageLine: "restore [-precaire]",
	Short:     "restore dependencies from the manifest",
	Long: `Restore vendor dependecies.

Flags:
		allow the use of insecure protocols.

`,
	Run: func(ctx *gb.Context, args []string) error {
		return restore(ctx)
	},
	AddFlags: addRestoreFlags,
}

func restore(ctx *gb.Context) error {
	m, err := vendor.ReadManifest(manifestFile(ctx))
	if err != nil {
		return fmt.Errorf("could not load manifest: %v", err)
	}

	for _, dep := range m.Dependencies {
		fmt.Printf("Getting %s\n", dep.Importpath)
		repo, _, err := vendor.DeduceRemoteRepo(dep.Importpath, insecure)
		if err != nil {
			return fmt.Errorf("Could not process dependency: %s", err)
		}
		wc, err := repo.Checkout("", "", dep.Revision)
		if err != nil {
			return fmt.Errorf("Could not retrieve dependency: %s", err)
		}
		dst := filepath.Join(ctx.Projectdir(), "vendor", "src", dep.Importpath)
		src := filepath.Join(wc.Dir(), dep.Path)

		if err := vendor.Copypath(dst, src); err != nil {
			return err
		}

		if err := wc.Destroy(); err != nil {
			return err
		}

	}
	return nil
}
