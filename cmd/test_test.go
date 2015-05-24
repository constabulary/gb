package cmd

import (
	"testing"
	"time"

	"github.com/constabulary/gb"
)

func TestTestPackage(t *testing.T) {
	gb.Verbose = false
	defer func() { gb.Verbose = false }()
	tests := []struct {
		pkg      string
		testArgs []string
		ldflags  string
		err      error
	}{
		{
			pkg: "a",
			err: nil,
		}, {
			pkg: "b",
			err: nil,
		}, {
			pkg: "c",
			err: nil,
		}, {
			pkg: "e",
			err: nil,
		}, {
			pkg: "cmd/f",
			err: nil,
		}, {
			pkg: "extest", // test external tests
			err: nil,
		}, {
			pkg: "g", // test that _test files can modify the internal package under test
			err: nil,
		}, {
			pkg:     "ldflags",
			ldflags: "-X ldflags.gitTagInfo banana -X ldflags.gitRevision f7926af2",
		}, {
			pkg: "cgotest",
		}, {
			pkg:      "testflags",
			testArgs: []string{"-debug"},
		}}

	for _, tt := range tests {
		ctx := testContext(t, gb.Ldflags(tt.ldflags))
		// TODO(dfc) can we resolve the duplication here ?
		pkg, err := ctx.ResolvePackageWithTests(tt.pkg)
		if err != nil {
			t.Errorf("ResolvePackage(%v): want %v, got %v", tt.pkg, tt.err, err)
			continue
		}
		if err := Test(tt.testArgs, pkg); err != tt.err {
			t.Errorf("Test(%v): want %v, got %v", tt.pkg, tt.err, err)
			time.Sleep(500 * time.Millisecond)
		}
		ctx.Destroy()
	}
}
