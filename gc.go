package gb

// gc toolchain

type gcToolchain struct {
	goos, goarch         string
	gc, cc, ld, as, pack string
}

type gcoption struct {
	goos, goarch string
}
