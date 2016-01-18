package gb

import "testing"

func TestStripext(t *testing.T) {
	tests := []struct {
		path, want string
	}{
		{"a.txt", "a"},
		{"a.a.txt", "a.a"},
		{"Makefile", "Makefile"},
		{"", ""},
		{"/", "/"},
	}

	for _, tt := range tests {
		got := stripext(tt.path)
		if got != tt.want {
			t.Errorf("stripext(%q): want: %v, got: %v", tt.path, tt.want, got)
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

		got, _ := relImportPath(tt.root, tt.path)
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
