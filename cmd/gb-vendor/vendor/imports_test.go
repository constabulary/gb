package vendor

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseImports(t *testing.T) {
	root := filepath.Join(getwd(t), "_testdata")

	got, err := ParseImports(root)
	if err != nil {
		t.Fatalf("ParseImports(%q): %v", root, err)
	}

	want := set("github.com/quux/flobble", "github.com/lypo/moopo", "github.com/hoo/wuu")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseImports(%q): want: %v, got %v", root, want, got)
	}
}

func TestFetchMetadata(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{{
		path: "golang.org/x/tools/cmd/godoc",
		want: `<!DOCTYPE html>
<html>
<head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
<meta name="go-import" content="golang.org/x/tools git https://go.googlesource.com/tools">
<meta name="go-source" content="golang.org/x/tools https://github.com/golang/tools/ https://github.com/golang/tools/tree/master{/dir} https://github.com/golang/tools/blob/master{/dir}/{file}#L{line}">
<meta http-equiv="refresh" content="0; url=https://godoc.org/golang.org/x/tools/cmd/godoc">
</head>
<body>
Nothing to see here; <a href="https://godoc.org/golang.org/x/tools/cmd/godoc">move along</a>.
</body>
</html>
`,
	}, {
		path: "gopkg.in/check.v1",
		want: `
<html>
<head>
<meta name="go-import" content="gopkg.in/check.v1 git https://gopkg.in/check.v1">
<meta name="go-source" content="gopkg.in/check.v1 _ https://github.com/go-check/check/tree/v1{/dir} https://github.com/go-check/check/blob/v1{/dir}/{file}#L{line}">
</head>
<body>
go get gopkg.in/check.v1
</body>
</html>
`,
	}}

	for _, tt := range tests {
		r, err := FetchMetadata(tt.path)
		if err != nil {
			t.Error(err)
			continue
		}
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, r); err != nil {
			t.Error(err)
			r.Close()
			continue
		}
		r.Close()
		got := buf.String()
		if got != tt.want {
			t.Errorf("FetchMetadata(%q): want %q, got %q", tt.path, tt.want, got)
		}
	}
}

func TestParseMetadata(t *testing.T) {
	tests := []struct {
		path       string
		importpath string
		vcs        string
		reporoot   string
	}{{
		path:       "golang.org/x/tools/cmd/godoc",
		importpath: "golang.org/x/tools",
		vcs:        "git",
		reporoot:   "https://go.googlesource.com/tools",
	}, {
		path:       "gopkg.in/check.v1",
		importpath: "gopkg.in/check.v1",
		vcs:        "git",
		reporoot:   "https://gopkg.in/check.v1",
	}, {
		path:       "gopkg.in/mgo.v2/bson",
		importpath: "gopkg.in/mgo.v2",
		vcs:        "git",
		reporoot:   "https://gopkg.in/mgo.v2",
	}}

	for _, tt := range tests {
		importpath, vcs, reporoot, err := ParseMetadata(tt.path)
		if err != nil {
			t.Error(err)
			continue
		}
		if importpath != tt.importpath || vcs != tt.vcs || reporoot != tt.reporoot {
			t.Errorf("ParseMetadata(%q): want %s %s %s, got %s %s %s ", tt.path, tt.importpath, tt.vcs, tt.reporoot, importpath, vcs, reporoot)
		}
	}
}

func getwd(t *testing.T) string {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return cwd
}
