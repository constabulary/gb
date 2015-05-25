package vendor

import (
	"reflect"
	"testing"
)

func TestRepositoryFromPath(t *testing.T) {
	tests := []struct {
		path string
		want Repository
		err  error
	}{{
		path: "github.com/pkg/sftp",
		want: &GitRepo{
			URL: "https://github.com/pkg/sftp",
		},
	},
	}

	for _, tt := range tests {
		got, err := RepositoryFromPath(tt.path)
		if (err == nil && reflect.DeepEqual(got, tt.want)) || err != tt.err {
			t.Errorf("RepositoryFromPath(%q): want %v, %v, got %v, %v", tt.path, tt.want, tt.err, got, err)
		}
	}
}
