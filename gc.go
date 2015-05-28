package gb

import "path/filepath"

// gc toolchain

type gcToolchain struct {
	goos, goarch         string
	gc, cc, ld, as, pack string
}

type gcoption struct {
	goos, goarch string
}

func (t *gcToolchain) Pack(pkg *Package, afiles ...string) error {
	args := []string{"r"}
	args = append(args, afiles...)
	dir := filepath.Dir(afiles[0])
	return pkg.run(dir, nil, t.pack, args...)
}

func (t *gcToolchain) compiler() string { return t.gc }
func (t *gcToolchain) linker() string   { return t.ld }
