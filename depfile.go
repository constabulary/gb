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
		if version, ok := kv["version"]; ok {
			if !re.MatchString(version) {
				return nil, errors.Errorf("%s: %q is not a valid SemVer 2.0.0 version", prefix, version)
			}
			root := filepath.Join(cachePath(), hash(prefix, version))
			dest := filepath.Join(root, "src", filepath.FromSlash(prefix))
			fi, err := os.Stat(dest)
			if err == nil {
				if !fi.IsDir() {
					return nil, errors.Errorf("%s is not a directory", dest)
				}
			}
			if err != nil {
				if !os.IsNotExist(err) {
					return nil, err
				}
				if err := fetchVersion(root, dest, prefix, version); err != nil {
					return nil, err
				}
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

		if tag, ok := kv["tag"]; ok {
			root := filepath.Join(cachePath(), hash(prefix, tag))
			dest := filepath.Join(root, "src", filepath.FromSlash(prefix))
			fi, err := os.Stat(dest)
			if err == nil {
				if !fi.IsDir() {
					return nil, errors.Errorf("%s is not a directory", dest)
				}
			}
			if err != nil {
				if !os.IsNotExist(err) {
					return nil, err
				}
				if err := fetchTag(root, dest, prefix, tag); err != nil {
					return nil, err
				}
			}
			i = &_importer{
				Importer: i,
				im: importer.Importer{
					Context: ic,
					Root:    root,
				},
			}
			debug.Debugf("Add importer for %q: %v", prefix+" "+tag, root)
		}
	}
	return i, nil
}

func fetchVersion(root, dest, prefix, version string) error {
	if !strings.HasPrefix(prefix, "github.com") {
		return errors.Errorf("unable to fetch %v", prefix)
	}

	fmt.Printf("fetching %v (%v)\n", prefix, version)

	rc, err := fetchRelease(prefix, "v"+version)
	if err != nil {
		return err
	}
	defer rc.Close()
	return unpackReleaseTarball(dest, rc)
}

func fetchTag(root, dest, prefix, tag string) error {
	if !strings.HasPrefix(prefix, "github.com") {
		return errors.Errorf("unable to fetch %v", prefix)
	}

	fmt.Printf("fetching %v (%v)\n", prefix, tag)

	rc, err := fetchRelease(prefix, tag)
	if err != nil {
		return err
	}
	defer rc.Close()
	return unpackReleaseTarball(dest, rc)
}

func unpackReleaseTarball(dest string, r io.Reader) error {
	gzr, err := gzip.NewReader(r)
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
		os.RemoveAll(tmpdir)
		return errors.Wrap(err, "cannot read download directory")
	}
	re := regexp.MustCompile(`\w+-\w+-[a-z0-9]+`)
	for _, dent := range dents {
		if re.MatchString(dent.Name()) {
			if err := os.Rename(filepath.Join(tmpdir, dent.Name()), dest); err != nil {
				os.RemoveAll(tmpdir)
				return errors.Wrap(err, "unable to rename final cache dir")
			}
			return nil
		}
	}
	os.RemoveAll(tmpdir)
	return errors.New("release directory not found in tarball")
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

func isDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}
