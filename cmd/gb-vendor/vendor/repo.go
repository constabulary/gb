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

	// URL returns the URL the clone was taken from. It should
	// only be called after Clone.
	URL() string
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
	ghregex = regexp.MustCompile(`^github.com/([A-Za-z0-9-._]+)/([A-Za-z0-9-._]+)(/.+)?`)
	bbregex = regexp.MustCompile(`^bitbucket.org/([A-Za-z0-9-._]+)/([A-Za-z0-9-._]+)(/.+)?`)
	lpregex = regexp.MustCompile(`^launchpad.net/([A-Za-z0-9-._]+)(/[A-Za-z0-9-._]+)?(/.+)?`)
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
		repo, err := Gitrepo(fmt.Sprintf("https://github.com/%s/%s", v[1], v[2]))
		return repo, v[3], err
	case bbregex.MatchString(path):
		v := bbregex.FindStringSubmatch(path)
		v = append(v, "")
		repo, err := Gitrepo(fmt.Sprintf("https://bitbucket.org/%s/%s", v[1], v[2]))
		if err == nil {
			return repo, v[3], nil
		}
		repo, err = Hgrepo(fmt.Sprintf("https://bitbucket.org/%s/%s", v[1], v[2]))
		if err == nil {
			return repo, v[3], nil
		}
		return nil, "", fmt.Errorf("unknown repository type")
	case lpregex.MatchString(path):
		v := lpregex.FindStringSubmatch(path)
		v = append(v, "", "")
		if v[2] == "" {
			// launchpad.net/project"
			repo, err := Bzrrepo(fmt.Sprintf("https://launchpad.net/%v", v[1]))
			return repo, "", err
		}
		// launchpad.net/project/series"
		repo, err := Bzrrepo(fmt.Sprintf("https://launchpad.net/%s/%s", v[1], v[2]))
		return repo, v[3], err
	default:
		// no idea, try to resolve as a vanity import
		importpath, vcs, reporoot, err := ParseMetadata(path)
		if err != nil {
			return nil, "", err
		}
		extra := path[len(importpath):]
		switch vcs {
		case "git":
			repo, err := Gitrepo(reporoot)
			return repo, extra, err
		case "hg":
			repo, err := Hgrepo(reporoot)
			return repo, extra, err
		case "bzr":
			repo, err := Bzrrepo(reporoot)
			return repo, extra, err
		default:
			return nil, "", fmt.Errorf("unknown repository type: %q", vcs)
		}
	}
}

// Gitrepo returns a Repository representing a remote git repository.
func Gitrepo(url string) (Repository, error) {
	if err := probeGitUrl(url); err != nil {
		return nil, err
	}
	return &gitrepo{
		url: url,
	}, nil
}

func probeGitUrl(url string) error {
	_, err := run("git", "ls-remote", "--exit-code", url, "HEAD")
	return err
}

// gitrepo is a git Repository.
type gitrepo struct {

	// remote repository url, see man 1 git-clone
	url string
}

func (g *gitrepo) URL() string {
	return g.url
}

func (g *gitrepo) Clone() (WorkingCopy, error) {
	dir, err := mktmp()
	if err != nil {
		return nil, err
	}

	if err := runOut(os.Stderr, "git", "clone", g.url, dir); err != nil {
		os.RemoveAll(dir)
		return nil, err
	}

	return &GitClone{
		Path: dir,
	}, nil
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

// Hgrepo returns a Repository representing a remote git repository.
func Hgrepo(url string) (Repository, error) {
	if err := probeHgUrl(url); err != nil {
		return nil, err
	}
	return &hgrepo{
		url: url,
	}, nil
}

func probeHgUrl(url string) error {
	_, err := run("hg", "identify", url)
	return err
}

// hgrepo is a Mercurial repo.
type hgrepo struct {

	// remote repository url, see man 1 hg
	url string
}

func (h *hgrepo) URL() string { return h.url }

func (h *hgrepo) Clone() (WorkingCopy, error) {
	dir, err := mktmp()
	if err != nil {
		return nil, err
	}

	if err := runOut(os.Stderr, "hg", "clone", h.url, dir); err != nil {
		return nil, err
	}

	return &HgClone{
		Path: dir,
	}, nil
}

// HgClone is a mercurial WorkingCopy.
type HgClone struct {
	Path string
}

func (h *HgClone) Dir() string { return h.Path }

func (h *HgClone) CheckoutBranch(branch string) error {
	_, err := run("hg", "--cwd", h.Path, "update", "-r", branch)
	return err
}

func (h *HgClone) CheckoutRevision(revision string) error {
	_, err := run("hg", "--cwd", h.Path, "update", "-r", revision)
	return err
}

func (h *HgClone) Revision() (string, error) {
	rev, err := run("hg", "--cwd", h.Path, "id", "-i")
	return strings.TrimSpace(string(rev)), err
}

func (h *HgClone) Branch() (string, error) {
	rev, err := run("hg", "--cwd", h.Path, "branch")
	return strings.TrimSpace(string(rev)), err
}

func (h *HgClone) Destroy() error {
	parent := filepath.Dir(h.Path)
	if err := os.RemoveAll(h.Path); err != nil {
		return err
	}
	return cleanPath(parent)
}

// Bzrrepo returns a Repository representing a remote bzr repository.
func Bzrrepo(url string) (Repository, error) {
	if err := probeBzrUrl(url); err != nil {
		return nil, err
	}
	return &bzrrepo{
		url: url,
	}, nil
}

func probeBzrUrl(url string) error {
	_, err := run("bzr", "info", url)
	return err
}

// bzrrepo is a bzr Repository.
type bzrrepo struct {

	// remote repository url
	url string
}

func (b *bzrrepo) URL() string {
	return b.url
}

func (b *bzrrepo) Clone() (WorkingCopy, error) {
	dir, err := mktmp()
	if err != nil {
		return nil, err
	}
	dir = filepath.Join(dir, "wc")
	if err := runOut(os.Stderr, "bzr", "branch", b.url, dir); err != nil {
		os.RemoveAll(dir)
		return nil, err
	}

	return &BzrClone{
		Path: dir,
	}, nil
}

// BzrClone is a bazaar WorkingCopy.
type BzrClone struct {
	Path string
}

func (b *BzrClone) Dir() string { return b.Path }

func (b *BzrClone) CheckoutBranch(branch string) error {
	// checkout branch is a noop for bzr
	return nil
}

func (b *BzrClone) CheckoutRevision(revision string) error {
	// checkout branch is a noop for bzr
	return nil
}

func (b *BzrClone) Revision() (string, error) {
	return "1", nil
}

func (b *BzrClone) Branch() (string, error) {
	return "master", nil
}

func (b *BzrClone) Destroy() error {
	parent := filepath.Dir(b.Path)
	if err := os.RemoveAll(b.Path); err != nil {
		return err
	}
	return cleanPath(parent)
}

func cleanPath(path string) error {
	if files, _ := ioutil.ReadDir(path); len(files) > 0 || filepath.Base(path) == "src" {
		return nil
	}
	parent := filepath.Dir(path)
	if err := os.RemoveAll(path); err != nil {
		return err
	}
	return cleanPath(parent)
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
