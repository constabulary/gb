package gb

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

	"github.com/constabulary/gb/internal/debug"
	"github.com/constabulary/gb/internal/depfile"
	"github.com/constabulary/gb/internal/importer"
	"github.com/constabulary/gb/internal/untar"
	"github.com/pkg/errors"
)

const semverRegex = `^([0-9]+)\.([0-9]+)\.([0-9]+)(?:(\-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+[0-9A-Za-z-\-\.]+)?$`

// addDepfileDeps inserts into the Context's importer list
// a set of importers for entries in the depfile.
func addDepfileDeps(ic *importer.Context, ctx *Context) (Importer, error) {
	i := Importer(new(nullImporter))
	df, err := readDepfile(ctx)
	if err != nil {
		if !os.IsNotExist(errors.Cause(err)) {
			return nil, errors.Wrap(err, "could not parse depfile")
		}
		debug.Debugf("no depfile, nothing to do.")
		return i, nil
	}
	re := regexp.MustCompile(semverRegex)
	for prefix, kv := range df {
		version, ok := kv["version"]
		if !ok {
			// TODO(dfc) return error when version key missing
			continue
		}
		if !re.MatchString(version) {
			return nil, errors.Errorf("%s: %q is not a valid SemVer 2.0.0 version", prefix, version)
		}
		root := filepath.Join(cachePath(), hash(prefix, version))
		if err := fetchIfMissing(root, prefix, version); err != nil {
			return nil, err
		}
		i = &_importer{
			Importer: i,
			im: importer.Importer{
				Context: ic,
				Root:    root,
			},
		}
		debug.Debugf("Add importer for %q: %v", prefix+" "+version, root)
	}
	return i, nil
}

func fetchIfMissing(root, prefix, version string) error {
	dest := filepath.Join(root, "src", filepath.FromSlash(prefix))
	_, err := os.Stat(dest)
	if err == nil {
		// not missing, nothing to do
		return nil
	}
	if !os.IsNotExist(err) {
		return errors.Wrap(err, "unexpected error stating cache dir")
	}
	if !strings.HasPrefix(prefix, "github.com") {
		return errors.Errorf("unable to fetch %v", prefix)
	}

	fmt.Printf("fetching %v (%v)\n", prefix, version)

	rc, err := fetchVersion(prefix, version)
	if err != nil {
		return err
	}
	defer rc.Close()

	gzr, err := gzip.NewReader(rc)
	if err != nil {
		return errors.Wrap(err, "unable to construct gzip reader")
	}

	parent, pkg := filepath.Split(dest)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return err
	}
	tmpdir, err := ioutil.TempDir(parent, "tmp")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpdir)

	tmpdir = filepath.Join(tmpdir, pkg)

	if err := untar.Untar(tmpdir, gzr); err != nil {
		return err
	}

	dents, err := ioutil.ReadDir(tmpdir)
	if err != nil {
		os.RemoveAll(root)
		return errors.Wrap(err, "cannot read download directory")
	}
	re := regexp.MustCompile(`\w+-\w+-[a-z0-9]+`)
	for _, dent := range dents {
		if re.MatchString(dent.Name()) {
			if err := os.Rename(filepath.Join(tmpdir, dent.Name()), dest); err != nil {
				os.RemoveAll(root)
				return errors.Wrap(err, "unable to rename final cache dir")
			}
			return nil
		}
	}
	os.RemoveAll(root)
	return errors.New("release directory not found in tarball")
}

func fetchVersion(prefix, version string) (io.ReadCloser, error) {
	return fetchRelease(prefix, "v"+version)
}

func fetchRelease(prefix, tag string) (io.ReadCloser, error) {
	const format = "https://api.github.com/repos/%s/tarball/%s"
	prefix = prefix[len("github.com/"):]
	url := fmt.Sprintf(format, prefix, tag)
	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch %q", url)
	}
	if resp.StatusCode != 200 {
		return nil, errors.Errorf("failed to fetch %q: expected 200, got %d", url, resp.StatusCode)
	}
	return resp.Body, nil
}

func readDepfile(ctx *Context) (map[string]map[string]string, error) {
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
