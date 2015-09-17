package gb

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRelativizeGcPaths(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("test needs current working directory to run: %v", err)
	}

	srcdir := filepath.Join(wd, "x/y")
	expected := []string{filepath.Join("x", "y", "a.go"), filepath.Join("x", "y", "b.go")}
	filenames := []string{"a.go", "b.go"}

	var gc gcToolchain
	gc.runOut = func(_ io.Writer, _ string, env []string, _ string, args ...string) error {
		// the filenames are the last arguments
		actual := args[len(args)-len(filenames):]
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("Source file paths were not properly relativized. Actual '%v', Expected: '%v'", actual, expected)
		}
		return nil
	}

	// we just need a fake package structure to pass through to gc.Gc
	c := testContext(t)
	p, err := c.ResolvePackage("a")
	if err != nil {
		t.Fatal(err)
	}
	gc.Gc(p, []string{}, "", srcdir, "", filenames)
}
