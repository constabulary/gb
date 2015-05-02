package gb

import "testing"

func TestBuild(t *testing.T) {
	Verbose = true
	defer func() { Verbose = false }()
	tests := []struct {
		pkg string
		err error
	}{{
		pkg: "a",
		err: nil,
	}, {
		pkg: "b", // actually command
		err: nil,
	}, {
		pkg: "c",
		err: nil,
	}, {
		pkg: "d.v1",
		err: nil,
	}}

	for _, tt := range tests {
		ctx := testContext(t)
		pkg, err := ctx.ResolvePackage(tt.pkg)
		if err != tt.err {
			t.Errorf("ctx.ResolvePackage(%v): want %v, got %v", tt.pkg, tt.err, err)
			continue
		}
		if err := Build(pkg); err != tt.err {
			t.Errorf("ctx.Build(%v): want %v, got %v", tt.pkg, tt.err, err)
		}
		ctx.Destroy()
	}
}
