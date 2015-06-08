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

// RemoteRepo describes a remote dvcs repository.
type RemoteRepo interface {

	// Checkout checks out the specific branch and revision
	// If branch is empty, the default branch for the underlying
	// VCS will be used. If revision is empty, the latest available
	// revision, taking into account branch, will be fetched.
	Checkout(branch, revision string) (WorkingCopy, error)

	// URL returns the URL the clone was taken from. It should
	// only be called after Clone.
	URL() string
}

// WorkingCopy represents a local copy of a remote dvcs repository.
type WorkingCopy interface {

	// Dir is the root of this working copy.
	Dir() string

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

// DeduceRemoteRepo takes a potential import path and returns a RemoteRepo
// representing the remote location of the source of an import path.
func DeduceRemoteRepo(path string) (RemoteRepo, string, error) {
	validimport := regexp.MustCompile(`^([A-Za-z0-9-]+)(.[A-Za-z0-9-]+)+(/.[A-Za-z0-9-_.]+)+$`)
	if !validimport.MatchString(path) {
		return nil, "", fmt.Errorf("%q is not a valid import path", path)
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

// Gitrepo returns a RemoteRepo representing a remote git repository.
func Gitrepo(url string) (RemoteRepo, error) {
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

// gitrepo is a git RemoteRepo.
type gitrepo struct {

	// remote repository url, see man 1 git-clone
	url string
}

func (g *gitrepo) URL() string {
	return g.url
}

func (g *gitrepo) Checkout(branch, revision string) (WorkingCopy, error) {
	dir, err := mktmp()
	if err != nil {
		return nil, err
	}

	args := []string{
		"clone",
		g.url,
		dir,
		"--single-branch",
	}
	if branch != "" {
		args = append(args, "--branch", branch)
	}

	if err := runOut(os.Stderr, "git", args...); err != nil {
		os.RemoveAll(dir)
		return nil, err
	}

	if revision != "" {
		if err := runOutPath(os.Stderr, dir, "git", "checkout", revision); err != nil {
			os.RemoveAll(dir)
			return nil, err
		}
	}

	return &GitClone{
		workingcopy{
			path: dir,
		},
	}, nil
}

type workingcopy struct {
	path string
}

func (w workingcopy) Dir() string { return w.path }

func (w workingcopy) Destroy() error {
	if err := os.RemoveAll(w.path); err != nil {
		return err
	}
	parent := filepath.Dir(w.path)
	return cleanPath(parent)
}

// GitClone is a git WorkingCopy.
type GitClone struct {
	workingcopy
}

func (g *GitClone) Revision() (string, error) {
	rev, err := runPath(g.path, "git", "rev-parse", "HEAD")
	return strings.TrimSpace(string(rev)), err
}

func (g *GitClone) Branch() (string, error) {
	rev, err := runPath(g.path, "git", "rev-parse", "--abbrev-ref", "HEAD")
	return strings.TrimSpace(string(rev)), err
}

// Hgrepo returns a RemoteRepo representing a remote git repository.
func Hgrepo(url string) (RemoteRepo, error) {
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

func (h *hgrepo) Checkout(branch, revision string) (WorkingCopy, error) {
	dir, err := mktmp()
	if err != nil {
		return nil, err
	}
	args := []string{
		"clone",
		h.url,
		dir,
	}

	if branch != "" {
		args = append(args, "--branch", branch)
	}
	if err := runOut(os.Stderr, "hg", args...); err != nil {
		os.RemoveAll(dir)
		return nil, err
	}
	if revision != "" {
		if err := runOut(os.Stderr, "hg", "--cwd", dir, "update", "-r", revision); err != nil {
			os.RemoveAll(dir)
			return nil, err
		}
	}

	return &HgClone{
		workingcopy{
			path: dir,
		},
	}, nil
}

// HgClone is a mercurial WorkingCopy.
type HgClone struct {
	workingcopy
}

func (h *HgClone) Revision() (string, error) {
	rev, err := run("hg", "--cwd", h.path, "id", "-i")
	return strings.TrimSpace(string(rev)), err
}

func (h *HgClone) Branch() (string, error) {
	rev, err := run("hg", "--cwd", h.path, "branch")
	return strings.TrimSpace(string(rev)), err
}

// Bzrrepo returns a RemoteRepo representing a remote bzr repository.
func Bzrrepo(url string) (RemoteRepo, error) {
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

// bzrrepo is a bzr RemoteRepo.
type bzrrepo struct {

	// remote repository url
	url string
}

func (b *bzrrepo) URL() string {
	return b.url
}

func (b *bzrrepo) Checkout(branch, revision string) (WorkingCopy, error) {
	dir, err := mktmp()
	if err != nil {
		return nil, err
	}
	wc := filepath.Join(dir, "wc")
	if err := runOut(os.Stderr, "bzr", "branch", b.url, wc); err != nil {
		os.RemoveAll(dir)
		return nil, err
	}

	return &BzrClone{
		workingcopy{
			path: dir,
		},
	}, nil
}

// BzrClone is a bazaar WorkingCopy.
type BzrClone struct {
	workingcopy
}

func (b *BzrClone) Revision() (string, error) {
	return "1", nil
}

func (b *BzrClone) Branch() (string, error) {
	return "master", nil
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

func runPath(path string, c string, args ...string) ([]byte, error) {
	var buf bytes.Buffer
	err := runOutPath(&buf, path, c, args...)
	return buf.Bytes(), err
}

func runOutPath(w io.Writer, path string, c string, args ...string) error {
	cmd := exec.Command(c, args...)
	cmd.Dir = path
	cmd.Stdout = w
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
