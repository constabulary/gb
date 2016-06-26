package main

import (
	"compress/gzip"
	"crypto/sha1"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd/gb/internal/depfile"
	"github.com/constabulary/gb/cmd/gb/internal/untar"
	"github.com/constabulary/gb/internal/debug"
	"github.com/constabulary/gb/internal/importer"
	"github.com/pkg/errors"
)

const semverRegex = `^([0-9]+)\.([0-9]+)\.([0-9]+)(?:(\-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+[0-9A-Za-z-\-\.]+)?$`

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

	re := regexp.MustCompile(semverRegex)
	for prefix, kv := range df {
		version, ok := kv["version"]
		if !ok {
			// TODO(dfc) return error when version key missing
			continue
		}
		if !re.MatchString(version) {
			fatalf("%s: %q is not a valid SemVer 2.0.0 version", prefix, version)
		}
		root := filepath.Join(cachePath(), hash(prefix, version))
		fetchIfMissing(root, prefix, version)
		im := importer.Importer{
			Context: ctx.Context, // TODO(dfc) this is a hack
			Root:    root,
		}
		debug.Debugf("Add importer for %q: %v", prefix+" "+version, im.Root)
		ctx.AddImporter(&im)
	}
}

func fetchIfMissing(root, prefix, version string) {
	dest := filepath.Join(root, "src", filepath.FromSlash(prefix))
	_, err := os.Stat(dest)
	if err == nil {
		// not missing, nothing to do
		return
	}
	if !os.IsNotExist(err) {
		fatalf("unexpected error stating cache dir: %v", err)
	}
	if !strings.HasPrefix(prefix, "github.com") {
		fatalf("unable to fetch %v", prefix)
	}

	fmt.Printf("fetching %v (%v)\n", prefix, version)

	rc := fetchVersion(prefix, version)
	defer rc.Close()

	gzr, err := gzip.NewReader(rc)
	if err != nil {
		fatalf("unable to construct gzip reader: %v", err)
	}

	parent, _ := filepath.Split(dest)
	mkdirall(parent)
	tmpdir := tempdir(parent)

	if err := untar.Untar(tmpdir, gzr); err != nil {
		fatalf("unable to untar: %v", err)
	}
	defer os.RemoveAll(tmpdir)

	dents, err := ioutil.ReadDir(tmpdir)
	if err != nil {
		os.RemoveAll(root)
		fatalf("cannot read download directory: %v", err)
	}
	re := regexp.MustCompile(`\w+-\w+-[a-z0-9]+`)
	for _, dent := range dents {
		if re.MatchString(dent.Name()) {
			if err := os.Rename(filepath.Join(tmpdir, dent.Name()), dest); err != nil {
				os.RemoveAll(root)
				fatalf("unable to rename final cache dir: %v", err)
			}
			return
		}
	}
	os.RemoveAll(root)
	fatalf("release directory not found in tarball")
}

func tempdir(parent string) string {
	path, err := ioutil.TempDir(parent, "tmp")
	if err != nil {
		fatalf("unable to create temporary dir: %v", err)
	}
	return path
}

func mkdirall(path string) {
	if err := os.MkdirAll(path, 0755); err != nil {
		fatalf("unable to create directory: %v", err)
	}
}

func fetchVersion(prefix, version string) io.ReadCloser {
	return fetchRelease(prefix, "v"+version)
}

func fetchRelease(prefix, tag string) io.ReadCloser {
	const format = "https://api.github.com/repos/%s/tarball/%s"
	prefix = prefix[len("github.com/"):]
	url := fmt.Sprintf(format, prefix, tag)
	resp, err := http.Get(url)
	if err != nil {
		fatalf("failed to fetch %q: %v", url, err)
	}
	if resp.StatusCode != 200 {
		fatalf("failed to fetch %q: expected 200, got %d", url, resp.StatusCode)
	}
	return resp.Body
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
