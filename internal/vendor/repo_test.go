package vendor

import (
	"fmt"
	"reflect"
	"testing"
)

func TestDeduceRemoteRepo(t *testing.T) {
	if testing.Short() {
		t.Skipf("skipping network tests in -short mode")
	}
	tests := []struct {
		path     string
		want     RemoteRepo
		extra    string
		err      error
		insecure bool
	}{{
		path: "",
		err:  fmt.Errorf(`"" is not a valid import path`),
	}, {
		path: "corporate",
		err:  fmt.Errorf(`"corporate" is not a valid import path`),
	}, {
		path: "github.com/cznic/b",
		want: &gitrepo{
			url: "https://github.com/cznic/b",
		},
	}, {
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
		/*
			bitbucket cannot maintain a stable ssh key across their app servers
			and this mucks up ci testing because mercurial does not have any
			way of unconditionally accepting new ssh keys for the host.
			Great work TEAM.
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
		*/
	}, {
		path: "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git",
		want: &gitrepo{
			url: "https://git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git",
		},
	}, {
		path: "git.apache.org/thrift.git/lib/go/thrift",
		want: &gitrepo{
			url: "https://git.apache.org/thrift.git",
		},
		extra: "/lib/go/thrift",
	}, {
		path: "gopkg.in/check.v1",
		want: &gitrepo{
			url: "https://gopkg.in/check.v1",
		},
		extra: "",
	}, {
		path: "goji.io",
		want: &gitrepo{
			url: "https://github.com/goji/goji",
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
		insecure: true,
	}, {
		path: "launchpad.net/gnuflag",
		want: &bzrrepo{
			url: "https://launchpad.net/gnuflag",
		},
	}, {
		path: "https://github.com/pkg/sftp",
		want: &gitrepo{
			url: "https://github.com/pkg/sftp",
		},
	}, {
		path: "git://github.com/pkg/sftp",
		want: &gitrepo{
			url: "git://github.com/pkg/sftp",
		},
		insecure: true,
	}}

	for _, tt := range tests {
		t.Logf("DeduceRemoteRepo(%q, %v)", tt.path, tt.insecure)
		got, extra, err := DeduceRemoteRepo(tt.path, tt.insecure)
		if !reflect.DeepEqual(err, tt.err) {
			t.Errorf("DeduceRemoteRepo(%q): want err: %v, got err: %v", tt.path, tt.err, err)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) || extra != tt.extra {
			t.Errorf("DeduceRemoteRepo(%q): want %#v, %v, got %#v, %v", tt.path, tt.want, tt.extra, got, extra)
		}
	}
}
