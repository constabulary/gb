package importer

import (
	"errors"
	"reflect"
	"runtime"
	"testing"
)

func TestGoodOSArch(t *testing.T) {
	var (
		thisOS   = runtime.GOOS
		thisArch = runtime.GOARCH
		otherOS  = func() string {
			if thisOS != "darwin" {
				return "darwin"
			}
			return "linux"
		}()
		otherArch = func() string {
			if thisArch != "amd64" {
				return "amd64"
			}
			return "386"
		}()
	)
	tests := []struct {
		name   string
		result bool
	}{
		{"file.go", true},
		{"file.c", true},
		{"file_foo.go", true},
		{"file_" + thisArch + ".go", true},
		{"file_" + otherArch + ".go", false},
		{"file_" + thisOS + ".go", true},
		{"file_" + otherOS + ".go", false},
		{"file_" + thisOS + "_" + thisArch + ".go", true},
		{"file_" + otherOS + "_" + thisArch + ".go", false},
		{"file_" + thisOS + "_" + otherArch + ".go", false},
		{"file_" + otherOS + "_" + otherArch + ".go", false},
		{"file_foo_" + thisArch + ".go", true},
		{"file_foo_" + otherArch + ".go", false},
		{"file_" + thisOS + ".c", true},
		{"file_" + otherOS + ".c", false},
	}

	for _, test := range tests {
		if goodOSArchFile(thisOS, thisArch, test.name, make(map[string]bool)) != test.result {
			t.Fatalf("goodOSArchFile(%q) != %v", test.name, test.result)
		}
	}
}

func TestExt(t *testing.T) {
	tests := []struct {
		name, want string
	}{
		{"file.go", ".go"},
		{"file", ""},
		{".go", ".go"},
	}

	for _, tt := range tests {
		got := ext(tt.name)
		if got != tt.want {
			t.Errorf("ext(%q): got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestSplitQuoted(t *testing.T) {
	tests := []struct {
		str  string
		want []string
		err  error
	}{{
		str: `a b:"c d" 'e''f'  "g\""`, want: []string{"a", "b:c d", "ef", `g"`},
	}, {
		str: `a b:"c d`, err: errors.New("unclosed quote"),
	}, {
		str: `a \`, err: errors.New("unfinished escaping"),
	}}

	for _, tt := range tests {
		got, err := splitQuoted(tt.str)
		if !reflect.DeepEqual(err, tt.err) {
			t.Errorf("splitQuoted(%q): got err %v, want err %v", tt.str, err, tt.err)
			continue
		}
		if err == nil && !reflect.DeepEqual(got, tt.want) {
			t.Errorf("splitQuoted(%q): got %v, want %v", tt.str, got, tt.want)
		}
	}
}

func TestCgoEnabled(t *testing.T) {
	tests := []struct {
		gohostos, gohostarch     string
		gotargetos, gotargetarch string
		want                     bool
	}{{
		"linux", "amd64", "linux", "amd64", true,
	}, {
		"linux", "amd64", "linux", "386", false,
	}}

	for _, tt := range tests {
		got := cgoEnabled(tt.gohostos, tt.gohostarch, tt.gotargetos, tt.gotargetarch)
		if got != tt.want {
			t.Errorf("cgoEnabled(%q, %q, %q, %q): got %v, want %v", tt.gohostos, tt.gohostarch, tt.gotargetos, tt.gotargetarch, got, tt.want)
		}
	}
}
