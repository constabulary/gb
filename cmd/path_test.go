package cmd

import "testing"

// disabled, FindProjectRoot uses os.Stat
func testFindProjectroot(t *testing.T) {
	tests := []struct {
		path   string
		gopath []string
		want   string
		err    error
	}{{
		path: "/home/foo/work/project/src",
		want: "/home/foo/work/project",
	}}

	for _, tt := range tests {
		got, err := FindProjectroot(tt.path)
		if got != tt.want || err != tt.err {
			t.Errorf("FindProjectroot(%v): want: %v, %v, got %v, %v", tt.path, tt.want, tt.err, got, err)
		}
	}
}

func TestRelImportPath(t *testing.T) {
	tests := []struct {
		root, path, want string
	}{
		{"/project/src", "a", "a"},
		// { "/project/src", "./a", "a"}, // TODO(dfc) this is relative
		// { "/project/src", "a/../b", "a"}, // TODO(dfc) so is this
	}

	for _, tt := range tests {
		got := relImportPath(tt.root, tt.path)
		if got != tt.want {
			t.Errorf("relImportPath(%q, %q): want: %v, got: %v", tt.root, tt.path, tt.want, got)
		}
	}
}

func TestIsRel(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{".", true},
		{"..", false},     // TODO(dfc) this is relative
		{"a/../b", false}, // TODO(dfc) this too
	}

	for _, tt := range tests {
		got := isRel(tt.path)
		if got != tt.want {
			t.Errorf("isRel(%q): want: %v, got: %v", tt.want, got)
		}
	}
}
