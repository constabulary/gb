package gb

// cgo support functions

// cgo returns a slice of post processed source files and a slice of 
// ObjTargets representing the result of compilation of the post .c
// output.
func cgo(pkg *Package) ([]ObjTarget, []string) {
	return nil, nil
}

