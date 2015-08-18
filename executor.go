package gb

import (
	"sync"
)

// Execute executes a tree of *Actions sequentually in depth first order.
func Execute(a *Action) error {
	seen := make(map[*Action]error)
	return execute(seen, a)
}

func execute(seen map[*Action]error, a *Action) error {
	// step 0, have we been here before
	if err, ok := seen[a]; ok {
		return err
	}

	// step 1, build all dependencies
	for _, d := range a.Deps {
		if err := execute(seen, d); err != nil {
			return err
		}
	}

	// step 2, now execute ourselves
	err := a.Run()
	seen[a] = err
	return err
}

// ExecuteConcurrent executes all actions in a tree concurrently.
// Each Action will wait until its dependant actions are complete.
func ExecuteConcurrent(a *Action, n int) error {
	var mu sync.Mutex // protects seen
	seen := make(map[*Action]chan error)

	get := func(result chan error) error {
		err := <-result
		result <- err
		return err
	}

	permits := make(chan bool, n)
	for i := 0; i < cap(permits); i++ {
		permits <- true
	}

	// wg tracks all the outstanding actions
	var wg sync.WaitGroup

	var execute func(map[*Action]chan error, *Action) chan error
	execute = func(seen map[*Action]chan error, a *Action) chan error {

		// step 0, have we seen this action before ?
		mu.Lock()
		if result, ok := seen[a]; ok {
			// yes! return the result channel so others can wait
			//  on our action
			mu.Unlock()
			return result
		}

		// step 1, we're the first to run this action
		result := make(chan error, 1)
		seen[a] = result
		mu.Unlock()

		// queue all dependant actions.
		var results []chan error
		for _, dep := range a.Deps {
			results = append(results, execute(seen, dep))
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			// wait for dependant actions
			for _, r := range results {
				if err := get(r); err != nil {
					result <- err
					return
				}
			}
			// wait for a permit and execute our action
			<-permits
			result <- a.Run()
			permits <- true
		}()

		return result

	}
	err := get(execute(seen, a))
	wg.Wait()
	return err
}
