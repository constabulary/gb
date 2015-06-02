package vendor

import (
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

	want := set("fmt", "github.com/quux/flobble")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseImports(%q): want: %v, got %v", root, want, got)
	}
}

func getwd(t *testing.T) string {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return cwd
}
