package depfile

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestParseKeyVal(t *testing.T) {
	tests := []struct {
		args []string
		want map[string]string
		err  error
	}{{
		args: []string{},          // handled by Parse
		want: map[string]string{}, // expected
	}, {
		args: []string{"version"},
		err:  fmt.Errorf("expected key=value pair, got %q", "version"),
	}, {
		args: []string{"=9.9.9"},
		err:  fmt.Errorf("expected key=value pair, missing key %q", "=9.9.9"),
	}, {
		args: []string{"version="},
		err:  fmt.Errorf("expected key=value pair, missing value %q", "version="),
	}, {
		args: []string{"version=1.2.3"},
		want: map[string]string{
			"version": "1.2.3",
		},
	}, {
		args: []string{"version=1.2.3", "version=2.4.5"},
		err:  fmt.Errorf("duplicate key=value pair, have \"version=1.2.3\" got %q", "version=2.4.5"),
	}, {
		args: []string{"version=1.2.3", "//", "comment"},
		err:  fmt.Errorf("expected key=value pair, got %q", "//"),
	}, {
		args: []string{"vcs=git", "version=1.2.3"},
		want: map[string]string{
			"version": "1.2.3",
			"vcs":     "git",
		},
	}}

	for _, tt := range tests {
		got, err := parseKeyVal(tt.args)
		if !reflect.DeepEqual(err, tt.err) {
			t.Errorf("parseKeyVal(%v): got %v, expected %v", tt.args, err, tt.err)
			continue
		}
		if err == nil && !reflect.DeepEqual(tt.want, got) {
			t.Errorf("parseKeyVal(%v): got %#v, expected %#v", tt.args, got, tt.want)
		}
	}
}

func TestParseLine(t *testing.T) {
	tests := []struct {
		line string
		name string
		kv   map[string]string
		err  error
	}{{
		line: "a", // no \n, sc.Text removes it
		err:  fmt.Errorf("a: expected key=value pair after name"),
	}, {
		line: "a\ta",
		err:  fmt.Errorf("a: expected key=value pair, got %q", "a"),
	}, {
		line: "a\tversion=7\t  ",
		name: "a",
		kv: map[string]string{
			"version": "7",
		},
	}}

	for _, tt := range tests {
		name, kv, err := parseLine(tt.line)
		if !reflect.DeepEqual(err, tt.err) {
			t.Errorf("parseLine(%q): got %v, expected %v", tt.line, err, tt.err)
			continue
		}
		if err == nil && !reflect.DeepEqual(tt.kv, kv) || name != tt.name {
			t.Errorf("parseLine(%q): got %s %#v, expected %s %#v", tt.line, name, kv, tt.name, tt.kv)
		}
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		c    string
		want map[string]map[string]string
		err  error
	}{{
		c:    "",                                 // empty
		want: make(map[string]map[string]string), // empty map, not nil
	}, {
		c:    "\n",                               // effectively empty
		want: make(map[string]map[string]string), // empty map, not nil
	}, {
		c:   "github.com/pkg/profile", // no \n
		err: fmt.Errorf("1: github.com/pkg/profile: expected key=value pair after name"),
	}, {
		c:   "github.com/pkg/profile\n",
		err: fmt.Errorf("1: github.com/pkg/profile: expected key=value pair after name"),
	}, {
		c: "github.com/pkg/profile version=1.2.3", // no \n
		want: map[string]map[string]string{
			"github.com/pkg/profile": {
				"version": "1.2.3",
			},
		},
	}, {
		c: "github.com/pkg/profile version=1.2.3\n",
		want: map[string]map[string]string{
			"github.com/pkg/profile": {
				"version": "1.2.3",
			},
		},
	}, {
		c: "// need to pin version\ngithub.com/pkg/profile version=1.2.3\n",
		want: map[string]map[string]string{
			"github.com/pkg/profile": {
				"version": "1.2.3",
			},
		},
	}, {
		c: "github.com/pkg/profile version=1.2.3\ngithub.com/pkg/sftp version=0.0.0",
		want: map[string]map[string]string{
			"github.com/pkg/profile": {
				"version": "1.2.3",
			},
			"github.com/pkg/sftp": {
				"version": "0.0.0",
			},
		},
	}, {
		c: `# some comment
github.com/pkg/profile version=0.1.0

; some other comment
// third kind of comment
 lines starting with blank lines are also ignored
github.com/pkg/sftp version=0.2.1
`,
		want: map[string]map[string]string{
			"github.com/pkg/profile": {
				"version": "0.1.0",
			},
			"github.com/pkg/sftp": {
				"version": "0.2.1",
			},
		},
	}}

	for _, tt := range tests {
		r := strings.NewReader(tt.c)
		got, err := Parse(r)
		if !reflect.DeepEqual(err, tt.err) {
			t.Errorf("Parse(%q): got %v, expected %v", tt.c, err, tt.err)
			continue
		}
		if err == nil && !reflect.DeepEqual(tt.want, got) {
			t.Errorf("Parse(%q): got %#v, expected %#v", tt.c, got, tt.want)
		}
	}
}

func TestSplitLine(t *testing.T) {
	tests := []struct {
		s    string
		want []string
	}{
		{s: "", want: nil},
		{s: "a", want: []string{"a"}},
		{s: " a", want: []string{"a"}},
		{s: "a ", want: []string{"a"}},
		{s: "a b", want: []string{"a", "b"}},
		{s: "a b", want: []string{"a", "b"}},
		{s: "a\tb", want: []string{"a", "b"}},
		{s: "a \tb", want: []string{"a", "b"}},
		{s: "\ta \tb ", want: []string{"a", "b"}},
	}

	for _, tt := range tests {
		got := splitLine(tt.s)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("splitLine(%q): got %#v, expected %#v", tt.s, got, tt.want)
		}
	}
}
