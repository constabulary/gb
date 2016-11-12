package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
	"github.com/constabulary/gb/internal/fileutils"
	"github.com/constabulary/gb/internal/vendor"
	"github.com/pkg/errors"
)

func addRestoreFlags(fs *flag.FlagSet) {
	fs.BoolVar(&insecure, "precaire", false, "allow the use of insecure protocols")
}

var cmdRestore = &cmd.Command{
	Name:      "restore",
	UsageLine: "restore [-precaire]",
	Short:     "restore dependencies from the manifest",
	Long: `Restore vendor dependencies.

Flags:
	-precaire
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
		return errors.Wrap(err, "could not load manifest")
	}

	errChan := make(chan error, 1)
	var wg sync.WaitGroup

	wg.Add(len(m.Dependencies))
	for _, dep := range m.Dependencies {
		go func(dep vendor.Dependency) {
			defer wg.Done()
			fmt.Printf("Getting %s\n", dep.Importpath)
			repo, _, err := vendor.DeduceRemoteRepo(dep.Importpath, insecure)
			if err != nil {
				errChan <- errors.Wrap(err, "could not process dependency")
				return
			}
			wc, err := repo.Checkout("", "", dep.Revision)
			if err != nil {
				errChan <- errors.Wrap(err, "could not retrieve dependency")
				return
			}
			dst := filepath.Join(ctx.Projectdir(), "vendor", "src", dep.Importpath)
			src := filepath.Join(wc.Dir(), dep.Path)

			if err := fileutils.Copypath(dst, src); err != nil {
				errChan <- err
				return
			}

			if err := wc.Destroy(); err != nil {
				errChan <- err
				return
			}
		}(dep)

	}

	wg.Wait()
	close(errChan)
	return <-errChan
}
