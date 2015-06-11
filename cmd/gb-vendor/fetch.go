package main

import (
	"flag"
	"fmt"
	"path/filepath"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
	"github.com/constabulary/gb/cmd/gb-vendor/vendor"
)

var (
	// gb vendor fetch command flags

	branch string

	// revision (commit)
	revision string

	tag string
)

func addFetchFlags(fs *flag.FlagSet) {
	fs.StringVar(&branch, "branch", "", "branch of the package")
	fs.StringVar(&revision, "revision", "", "revision of the package")
	fs.StringVar(&tag, "tag", "", "tag of the package")
}

var cmdFetch = &cmd.Command{
	Name:      "fetch",
	UsageLine: "fetch [-branch branch | -revision rev | -tag tag] importpath",
	Short:     "fetch a remote dependency",
	Long: `fetch vendors the upstream import path.

Flags:
	-branch branch
		fetch from the name branch. If not supplied the default upstream
		branch will be used.
	-tag tag
		fetch the specified tag. If not supplie the default upstream 
		branch will be used.
	-revision rev
		fetch the specific revision from the branch (if supplied). If no
		revision supplied, the latest available will be supplied.

`,
	Run: func(ctx *gb.Context, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("fetch: import path missing")
		}
		path := args[0]

		m, err := vendor.ReadManifest(manifestFile(ctx))
		if err != nil {
			return fmt.Errorf("could not load manifest: %v", err)
		}

		repo, extra, err := vendor.DeduceRemoteRepo(path)
		if err != nil {
			return err
		}

		if m.HasImportpath(path) {
			return fmt.Errorf("%s is already vendored", path)
		}

		wc, err := repo.Checkout(branch, tag, revision)
		if err != nil {
			return err
		}

		rev, err := wc.Revision()
		if err != nil {
			return err
		}

		branch, err := wc.Branch()
		if err != nil {
			return err
		}

		dep := vendor.Dependency{
			Importpath: path,
			Repository: repo.URL(),
			Revision:   rev,
			Branch:     branch,
			Path:       extra,
		}

		if err := m.AddDependency(dep); err != nil {
			return err
		}

		dst := filepath.Join(ctx.Projectdir(), "vendor", "src", dep.Importpath)
		src := filepath.Join(wc.Dir(), dep.Path)

		if err := vendor.Copypath(dst, src); err != nil {
			return err
		}

		if err := vendor.WriteManifest(manifestFile(ctx), m); err != nil {
			return err
		}
		return wc.Destroy()
	},
	AddFlags: addFetchFlags,
}
