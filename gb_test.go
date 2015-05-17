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
