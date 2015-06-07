package vendor

import (
	"reflect"
	"testing"
)

func TestRepositoryFromPath(t *testing.T) {
	tests := []struct {
		path  string
		want  Repository
		extra string
		err   error
	}{{
		path: "github.com/pkg/sftp",
		want: &gitrepo{
			url: "https://github.com/pkg/sftp",
		},
	}, {
		path: "github.com/pkg/sftp/examples/gsftp",
		want: &gitrepo{
			url: "https://github.com/pkg/sftp",
		},
		extra: "/examples/gsftp",
	}, {
		path: "github.com/coreos/go-etcd",
		want: &gitrepo{
			url: "https://github.com/coreos/go-etcd",
		},
	}, {
		path: "bitbucket.org/davecheney/gitrepo/cmd/main",
		want: &gitrepo{
			url: "https://bitbucket.org/davecheney/gitrepo",
		},
		extra: "/cmd/main",
	}, {
		path: "bitbucket.org/davecheney/hgrepo/cmd/main",
		want: &hgrepo{
			url: "https://bitbucket.org/davecheney/hgrepo",
		},
		extra: "/cmd/main",
	}, {
		path: "gopkg.in/check.v1",
		want: &gitrepo{
			url: "https://gopkg.in/check.v1",
		},
		extra: "",
	}, {
		path: "golang.org/x/tools/go/vcs",
		want: &gitrepo{
			url: "https://go.googlesource.com/tools",
		},
		extra: "/go/vcs",
	}, {
		path: "labix.org/v2/mgo",
		want: &bzrrepo{
			url: "https://launchpad.net/mgo/v2",
		},
	}, {
		path: "launchpad.net/gnuflag",
		want: &bzrrepo{
			url: "https://launchpad.net/gnuflag",
		},
	}}

	for _, tt := range tests {
		got, extra, err := RepositoryFromPath(tt.path)
		if !reflect.DeepEqual(got, tt.want) || extra != tt.extra || err != tt.err {
			t.Errorf("RepositoryFromPath(%q): want %#v, %v, %v, got %#v, %v, %v", tt.path, tt.want, tt.extra, tt.err, got, extra, err)
		}
	}
}
