package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
	"github.com/constabulary/gb/internal/fileutils"
	"github.com/constabulary/gb/internal/vendor"
	"github.com/pkg/errors"
)

var threadCount int

func addRestoreFlags(fs *flag.FlagSet) {
	fs.BoolVar(&insecure, "precaire", false, "allow the use of insecure protocols")
	fs.IntVar(&threadCount, "threads", runtime.GOMAXPROCS(-1), "thread number limit")
}

var cmdRestore = &cmd.Command{
	Name:      "restore",
	UsageLine: "restore [-precaire -threads]",
	Short:     "restore dependencies from the manifest",
	Long: `Restore vendor dependencies.

Flags:
	-precaire
		allow the use of insecure protocols.
	-threads int
		thread usage limit (default to cpu number)

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

	var (
		counter     int32
		workerGroup sync.WaitGroup
		depCount    = len(m.Dependencies)
		errChan     = make(chan error, threadCount)
		depChan     = make(chan vendor.Dependency)
		stopChan    = make(chan struct{})
		doneChan    = make(chan struct{})
	)

	fmt.Printf("Need install %d dependencies\n", depCount)

	workerGroup.Add(threadCount)
	go func() {
		defer close(errChan)
		defer close(doneChan)
		workerGroup.Wait()
	}()

	for i := 0; i < threadCount; i++ {
		go func() {
			defer workerGroup.Done()

			for dep := range depChan {
				fmt.Printf("[%v/%d] Getting %s\n", atomic.AddInt32(&counter, 1), depCount, dep.Importpath)

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
			}
		}()
	}

	go func() {
		defer close(depChan)

		for _, dep := range m.Dependencies {
			select {
			case <-stopChan:
				return
			case depChan <- dep:
			}
		}
	}()

	// wait error
	err = <-errChan
	// stop workers after first error
	close(stopChan)
	// exit after all goroutines have finished
	<-doneChan

	return err
}
