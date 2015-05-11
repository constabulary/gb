package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type context struct {
	srcdirs, allpkgs []string
}

func (c *context) Srcdirs() []string {
	return c.srcdirs
}

func (c *context) AllPackages(pattern string) []string {
	return c.allpkgs
}

func testdata(args ...string) string {
	cwd, _ := os.Getwd()
	return filepath.Join(append([]string{cwd, "testdata"}, args...)...)
}

// l constructs a []string
func l(args ...string) []string {
	return args
}

// p constructs a path
func p(args ...string) string {
	return filepath.Join(args...)
}

func TestImportPaths(t *testing.T) {
	var tests = []struct {
		ctx  context
		cwd  string
		args []string
		want []string
	}{
		{
			ctx: context{
				srcdirs: l(testdata("src")),
				allpkgs: l("a", "b", "c", p("c", "d")),
			},
			cwd:  testdata("src"),
			args: nil,
			want: l("a", "b", "c", p("c", "d")),
		}, {
			ctx: context{
				srcdirs: l(testdata("src")),
				allpkgs: l("a", "b", "c", p("c", "d")),
			},
			cwd:  testdata("src"),
			args: l("..."),
			want: l("a", "b", "c", p("c", "d")),
		}, {
			ctx: context{
				srcdirs: l(testdata("src")),
				allpkgs: l("c", p("c", "d")),
			},
			cwd:  testdata("src", "c"),
			args: nil,
			want: l("c", p("c", "d")),
		}, {
			ctx: context{
				srcdirs: l(testdata("src")),
				allpkgs: l("a", "b", "c", p("c", "d")),
			},
			cwd:  testdata("src"),
			args: l("c"),
			want: l("c"),
		}, {
			ctx: context{
				srcdirs: l(testdata("src")),
				allpkgs: l("a", "b", "c", p("c", "d")),
			},
			cwd:  testdata("src"),
			args: l("c", "b"),
			want: l("c", "b"),
		}, {
			ctx: context{
				srcdirs: l(testdata("src")),
				allpkgs: l("c", p("c", "d")),
			},
			cwd:  testdata("src"),
			args: l("c/..."),
			want: l("c", p("c", "d")),
		},
	}
	for _, tt := range tests {
		got := ImportPaths(&tt.ctx, tt.cwd, tt.args)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("ImportPaths(%v): got %v, want %v", tt.args, got, tt.want)
		}
	}
}
