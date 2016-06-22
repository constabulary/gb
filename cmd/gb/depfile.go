package main

import (
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd/gb/internal/depfile"
	"github.com/constabulary/gb/internal/debug"
	"github.com/constabulary/gb/internal/importer"
	"github.com/pkg/errors"
)

// addDepfileDeps inserts into the Context's importer list
// a set of importers for entries in the depfile.
func addDepfileDeps(ctx *gb.Context) {
	df, err := readDepfile(ctx)
	if err != nil {
		if !os.IsNotExist(errors.Cause(err)) {
			fatalf("could not parse depfile: %v", err)
		}
		debug.Debugf("no depfile, nothing to do.")
		return
	}
	for prefix, kv := range df {
		version, ok := kv["version"]
		if !ok {
			// TODO(dfc) return error when version key missing
			continue
		}
		root := filepath.Join(cachePath(), hash(prefix, version))
		im := importer.Importer{
			Context: ctx.Context, // TODO(dfc) this is a hack
			Root:    root,
		}
		debug.Debugf("Add importer for %q: %v", prefix+" "+version, im.Root)
		ctx.AddImporter(&im)
	}
}

func readDepfile(ctx *gb.Context) (map[string]map[string]string, error) {
	file := filepath.Join(ctx.Projectdir(), "depfile")
	debug.Debugf("loading depfile at %q", file)
	return depfile.ParseFile(file)
}

func hash(arg string, args ...string) string {
	h := sha1.New()
	io.WriteString(h, arg)
	for _, arg := range args {
		io.WriteString(h, arg)
	}
	return fmt.Sprintf("%x", string(h.Sum(nil)))
}

func cachePath() string {
	return filepath.Join(gbhome(), "cache")
}

func gbhome() string {
	return envOr("GB_HOME", filepath.Join(envOr("HOME", "/tmp"), ".gb"))
}

func envOr(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		v = def
	}
	return v
}
