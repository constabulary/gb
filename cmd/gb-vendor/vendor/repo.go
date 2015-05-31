package vendor

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// gb-vendor repo support

// Repository describes a remote dvcs repository.
type Repository interface {

	// Clone fetches the source of the remote repository.
	Clone() (WorkingCopy, error)
}

// WorkingCopy represents a local copy of a remote dvcs repository.
type WorkingCopy interface {

	// Dir is the root of this working copy.
	Dir() string

	// Checks out specific branch of this working copy.
	CheckoutBranch(string) error

	// Checks out specific revision of this working copy.
	CheckoutRevision(string) error

	// Revision returns the revision of this working copy.
	Revision() (string, error)

	// Branch returns the branch to which this working copy belongs.
	Branch() (string, error)

	// Destroy removes the working copy and cleans path to the working copy.
	Destroy() error
}

var (
	ghregex     = regexp.MustCompile(`^github.com/([A-Za-z0-9-._]+)/([A-Za-z0-9-._]+)(/.+)?`)
	golangregex = regexp.MustCompile(`^golang.org/x/([A-Za-z0-9-._]+)(/.+)?`)
)

// RepositoryFromPath attempts to deduce a Repository from an import path.
// If there are additional path items remaining then they will be returned.
func RepositoryFromPath(path string) (Repository, string, error) {
	if strings.Contains(path, "//:") {
		return nil, path, fmt.Errorf("path must not be a url")
	}

	switch {
	case ghregex.MatchString(path):
		v := ghregex.FindStringSubmatch(path)
		v = append(v, "")
		return &GitRepo{
			URL: fmt.Sprintf("https://github.com/%s/%s", v[1], v[2]),
		}, v[3], nil
	case golangregex.MatchString(path):
		v := golangregex.FindStringSubmatch(path)
		v = append(v, "")
		return &GitRepo{
			URL: fmt.Sprintf("https://go.googlesource.com/%s", v[1]),
		}, v[2], nil
	default:
		return nil, path, fmt.Errorf("unknown repository type")
	}
}

// GitRepo is git Repository.
type GitRepo struct {

	// remote repository url, see man 1 git-clone
	URL string
}

// GitClone is a git WorkingCopy.
type GitClone struct {
	Path string
}

func (g *GitClone) Dir() string { return g.Path }

func (g *GitClone) CheckoutBranch(branch string) error {
	_, err := run("git", "-C", g.Path, "checkout", "-b", branch, "origin/"+branch)
	return err
}

func (g *GitClone) CheckoutRevision(revision string) error {
	_, err := run("git", "-C", g.Path, "checkout", revision)
	return err
}

func (g *GitClone) Revision() (string, error) {
	rev, err := run("git", "-C", g.Path, "rev-parse", "HEAD")
	return strings.TrimSpace(string(rev)), err
}

func (g *GitClone) Branch() (string, error) {
	rev, err := run("git", "-C", g.Path, "rev-parse", "--abbrev-ref", "HEAD")
	return strings.TrimSpace(string(rev)), err
}

func (g *GitClone) Destroy() error {
	parent := filepath.Dir(g.Path)
	if err := os.RemoveAll(g.Path); err != nil {
		return err
	}
	return cleanPath(parent)
}

func (g *GitRepo) Clone() (WorkingCopy, error) {
	dir, err := mktmp()
	if err != nil {
		return nil, err
	}

	if err := runOut(os.Stderr, "git", "clone", g.URL, dir); err != nil {
		return nil, err
	}

	return &GitClone{
		Path: dir,
	}, nil
}

func cleanPath(path string) error {
	if files, _ := ioutil.ReadDir(path); len(files) > 0 || filepath.Base(path) == "src" {
		return nil
	} else {
		parent := filepath.Dir(path)
		if err := os.RemoveAll(path); err != nil {
			return err
		}
		return cleanPath(parent)
	}

}

func mkdir(path string) error {
	return os.MkdirAll(path, 0755)
}

func mktmp() (string, error) {
	return ioutil.TempDir("", "gb-vendor-")
}

func run(c string, args ...string) ([]byte, error) {
	var buf bytes.Buffer
	err := runOut(&buf, c, args...)
	return buf.Bytes(), err
}

func runOut(w io.Writer, c string, args ...string) error {
	cmd := exec.Command(c, args...)
	cmd.Stdout = w
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
