package gb

import (
	"testing"
	"time"
)

func TestTestPackage(t *testing.T) {
	Verbose = false
	defer func() { Verbose = false }()
	tests := []struct {
		pkg string
		err error
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
		}}

	for _, tt := range tests {
		ctx := testContext(t)
		pkg, err := ctx.ResolvePackage(tt.pkg)
		if err != nil {
			t.Errorf("ResolvePackage(%v): want %v, got %v", tt.pkg, tt.err, err)
			continue
		}
		if err := Test(pkg); err != tt.err {
			t.Errorf("Test(%v): want %v, got %v", tt.pkg, tt.err, err)
			time.Sleep(500 * time.Millisecond)
		}
		ctx.Destroy()
	}
}
