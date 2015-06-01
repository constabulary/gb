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
		want: &GitRepo{
			url: "https://github.com/pkg/sftp",
		},
	}, {
		path: "github.com/pkg/sftp/examples/gsftp",
		want: &GitRepo{
			url: "https://github.com/pkg/sftp",
		},
		extra: "/examples/gsftp",
	}, {
		path: "github.com/coreos/go-etcd",
		want: &GitRepo{
			url: "https://github.com/coreos/go-etcd",
		},
	}, {
		path: "bitbucket.org/StephaneBunel/xxhash-go",
		want: &MultiRepo{
			remotes: []Repository{
				&HgRepo{url: "https://bitbucket.org/StephaneBunel/xxhash-go"},
				&GitRepo{url: "https://bitbucket.org/StephaneBunel/xxhash-go"},
			},
		},
	}, {
		path: "bitbucket.org/user/project/sub/directory",
		want: &MultiRepo{
			remotes: []Repository{
				&HgRepo{url: "https://bitbucket.org/user/project"},
				&GitRepo{url: "https://bitbucket.org/user/project"},
			},
		},
		extra: "/sub/directory",
	}, {
		path: "gopkg.in/pg.v3",
		want: &GitRepo{
			url: "https://gopkg.in/pg.v3",
		},
	}, {
		path: "gopkg.in/user/pkg.v1",
		want: &GitRepo{
			url: "https://gopkg.in/user/pkg.v1",
		},
	}}

	for _, tt := range tests {
		got, extra, err := RepositoryFromPath(tt.path)
		if !reflect.DeepEqual(got, tt.want) || extra != tt.extra || err != tt.err {
			t.Errorf("RepositoryFromPath(%q): want %v, %v, %v, got %v, %v, %v", tt.path, tt.want, tt.extra, tt.err, got, extra, err)
		}
	}
}
