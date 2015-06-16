package gb

type Srcdir struct {
	// Root is the root directory of this Srcdir.
	Root string

	// Prefix is an optional import path prefix applied
	// to any package resolved via this Srcdir.
	Prefix string
}
