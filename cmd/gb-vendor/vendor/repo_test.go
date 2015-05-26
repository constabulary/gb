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
			URL: "https://github.com/pkg/sftp",
		},
	}, {
		path: "github.com/pkg/sftp/examples/gsftp",
		want: &GitRepo{
			URL: "https://github.com/pkg/sftp",
		},
		extra: "/examples/gsftp",
	}}

	for _, tt := range tests {
		got, extra, err := RepositoryFromPath(tt.path)
		if !reflect.DeepEqual(got, tt.want) || extra != tt.extra || err != tt.err {
			t.Errorf("RepositoryFromPath(%q): want %v, %v, %v, got %v, %v, %v", tt.path, tt.want, tt.extra, tt.err, got, extra, err)
		}
	}
}
