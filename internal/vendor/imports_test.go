package vendor

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseImports(t *testing.T) {
	root := filepath.Join(getwd(t), "_testdata", "src")

	got, err := ParseImports(root)
	if err != nil {
		t.Fatalf("ParseImports(%q): %v", root, err)
	}

	want := set("fmt", "github.com/quux/flobble", "github.com/lypo/moopo", "github.com/hoo/wuu")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseImports(%q): want: %v, got %v", root, want, got)
	}
}

func TestParseMetaRemoteImportPaths(t *testing.T) {
	tests := []struct {
		input string
		want  []metaImport
	}{
		// The "meta" element has a start tag, but no end tag.
		{`<meta name="go-import" content="golang.org/x/tools git https://go.googlesource.com/tools">`,
			[]metaImport{{"golang.org/x/tools", "git", "https://go.googlesource.com/tools"}}},
		// The parser tolerates unquoted XML attribute values, but note that the CDATA section is not terminated properly.
		{`<!doctype html><title>Page Not Found</title>
<meta name="go-import" content="golang.org/x/tools git https://go.googlesource.com/tools">
<meta name=go-import content="chitin.io/chitin git https://github.com/chitin-io/chitin">
<![CDATA[...]`,
			[]metaImport{
				{"golang.org/x/tools", "git", "https://go.googlesource.com/tools"},
				{"chitin.io/chitin", "git", "https://github.com/chitin-io/chitin"}}},
	}
	for _, tt := range tests {
		got, err := parseMetaGoImports(strings.NewReader(tt.input))
		if err != nil {
			t.Errorf("parseMetaGoImports(%q): %v", tt.input, err)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("parseMetaGoImports(%q): want %v, got %v", tt.input, tt.want, got)
		}
	}
}

func TestFetchMetadata(t *testing.T) {
	if testing.Short() {
		t.Skipf("skipping network tests in -short mode")
	}
	type testParams struct {
		path     string
		want     string
		insecure bool
	}
	tests := []testParams{{
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
		r, err := FetchMetadata(tt.path, tt.insecure)
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

	// Test for error catch.
	errTests := []testParams{{
		path:     "any.inaccessible.server/the.project",
		want:     `unable to determine remote metadata protocol: failed to access url "http://any.inaccessible.server/the.project?go-get=1"`,
		insecure: true,
	}, {
		path:     "any.inaccessible.server/the.project",
		want:     `unable to determine remote metadata protocol: failed to access url "https://any.inaccessible.server/the.project?go-get=1"`,
		insecure: false,
	}}

	for _, ett := range errTests {
		r, err := FetchMetadata(ett.path, ett.insecure)
		if err == nil {
			t.Errorf("Access to url %q without any error, but the error should be happen.", ett.path)
			if r != nil {
				r.Close()
			}
			continue
		}
		got := err.Error()
		if got != ett.want {
			t.Errorf("FetchMetadata(%q): want %q, got %q", ett.path, ett.want, got)
		}
	}
}

func TestParseMetadata(t *testing.T) {
	if testing.Short() {
		t.Skipf("skipping network tests in -short mode")
	}
	tests := []struct {
		path       string
		importpath string
		vcs        string
		reporoot   string
		insecure   bool
		err        error
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
		//	}, {
		//		path: "speter.net/go/exp",
		//		err:  fmt.Errorf("go-import metadata not found"),
	}}

	for _, tt := range tests {
		importpath, vcs, reporoot, err := ParseMetadata(tt.path, tt.insecure)
		if !reflect.DeepEqual(err, tt.err) {
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
