package vendor

// union returns the union of a and b.
func union(a, b map[string]bool) map[string]bool {
	r := make(map[string]bool)
	for k := range a {
		r[k] = true
	}
	for k := range b {
		r[k] = true
	}
	return r
}

// intersection returns the intersection of a and b.
func intersection(a, b map[string]bool) map[string]bool {
	r := make(map[string]bool)
	for k := range a {
		if b[k] {
			r[k] = true
		}
	}
	return r
}

// difference returns the difference of a and b.
func difference(a, b map[string]bool) map[string]bool {
	r := make(map[string]bool)
	for k := range a {
		if !b[k] {
			r[k] = true
		}
	}
	for k := range b {
		if !a[k] {
			r[k] = true
		}
	}
	return r
}
