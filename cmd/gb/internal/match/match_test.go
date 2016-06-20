package match

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestImportPaths(t *testing.T) {
	tests := []struct {
		args []string
		want []string
	}{{
		nil,
		[]string{"cmd", "cmd/main", "github.com", "github.com/foo", "github.com/foo/bar", "github.com/quxx", "github.com/quxx/bar"},
	}, {
		[]string{},
		[]string{"cmd", "cmd/main", "github.com", "github.com/foo", "github.com/foo/bar", "github.com/quxx", "github.com/quxx/bar"},
	}, {
		[]string{"."},
		[]string{"."},
	}, {
		[]string{".."},
		[]string{".."},
	}, {
		[]string{"./."},
		[]string{"."},
	}, {
		[]string{"..."},
		[]string{"cmd", "cmd/main", "github.com", "github.com/foo", "github.com/foo/bar", "github.com/quxx", "github.com/quxx/bar"},
	}, {
		[]string{".../bar"},
		[]string{"github.com/foo/bar", "github.com/quxx/bar"},
	}, {
		[]string{"cmd"},
		[]string{"cmd"},
	}, {
		[]string{"cmd/go"},
		[]string{"cmd/go"},
	}, {
		[]string{"cmd/main"},
		[]string{"cmd/main"},
	}, {
		[]string{"cmd/..."},
		[]string{"cmd", "cmd/main"},
	}, {
		[]string{"nonexist"},
		[]string{"nonexist"}, // passed through because there is no wildcard
	}}

	for _, tt := range tests {
		const srcdir = "_testdata/a/src"
		const cwd = "_testdata/a"
		got := ImportPaths(srcdir, cwd, tt.args)
		if !reflect.DeepEqual(tt.want, got) {
			t.Errorf("ImportPaths(%q): got %q, want %q", tt.args, got, tt.want)
		}
	}
}

func TestMatchPackages(t *testing.T) {
	tests := []struct {
		pattern string
		want    []string
	}{{
		"",
		nil,
	}, {
		"github.com/foo",
		[]string{
			"github.com/foo",
		},
	}, {
		"github.com/foo/...",
		[]string{
			"github.com/foo",
			"github.com/foo/bar",
		},
	}, {
		"github.com",
		[]string{
			"github.com",
		},
	}, {
		"github.com/...",
		[]string{
			"github.com",
			"github.com/foo",
			"github.com/foo/bar",
			"github.com/quxx",
			"github.com/quxx/bar",
		},
	}}

	for _, tt := range tests {
		srcdir := "_testdata/a/src"
		got, err := matchPackages(srcdir, tt.pattern)
		if err != nil {
			t.Error(err)
			continue
		}
		if !reflect.DeepEqual(tt.want, got) {
			t.Errorf("matchPackagesInSrcDir(%q, ..., %q): got %q, want %q", srcdir, tt.pattern, got, tt.want)
		}
	}
}

func TestIsLocalImport(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{".", true},
		{"", false},
		{"..", true},
		{"a/..", false},
		{"../a", true},
		{"./a", true},
	}

	for _, tt := range tests {
		got := isLocalImport(tt.path)
		if got != tt.want {
			t.Errorf("isLocalImportPath(%q): got: %v, want: %v", tt.path, got, tt.want)
		}
	}
}

func TestSkipElem(t *testing.T) {
	tests := []struct {
		elem string
		want bool
	}{
		{"", false},
		{".", true},
		{"..", true},
		{"a", false},
		{".a", true},
		{"a.", false},
		{"_", true},
		{"_a", true},
		{"a_", false},
		{"a", false},
		{"testdata", true},
		{"_testdata", true},
		{".testdata", true},
		{"testdata_", false},
	}

	for _, tt := range tests {
		got := skipElem(tt.elem)
		if got != tt.want {
			t.Errorf("skipElem(%q): got: %v, want: %v", tt.elem, got, tt.want)
		}
	}
}

func abs(t *testing.T, path string) string {
	path, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func absv(t *testing.T, paths ...string) []string {
	for i := range paths {
		paths[i] = abs(t, paths[i])
	}
	return paths
}
