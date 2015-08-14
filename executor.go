package gb

// Execute executes a tree of *Actions sequentually in depth first order.
func Execute(a *Action) error {
	seen := make(map[*Action]bool)
	return execute(seen, a)
}

func execute(seen map[*Action]bool, a *Action) error {
	// step 0, have we been here before
	if seen[a] {
		return nil
	}

	// step 1, build all dependencies
	for _, d := range a.Deps {
		if err := execute(seen, d); err != nil {
			return err
		}
	}

	// step 2, now execute ourselves
	seen[a] = true
	return a.Run()
}

// ExecuteConcurrent executes all actions in a tree concurrently.
// Each Action will wait until its dependant actions are complete.
func ExecuteConcurrent(a *Action) error {
	return Execute(a) // ha!
}
