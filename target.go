package gb

import "fmt"

// A Target is a placeholder for work which is completed asyncronusly.
type Target interface {
	// Result returns the result of the work as an error, or nil if the work
	// was performed successfully.
	// Implementers must observe these invariants
	// 1. There may be multiple concurrent callers to Result, or Result may
	//    be called many times in sequence, it must always return the same
	// 2. Result blocks until the work has been performed.
	Result() error
}

type target struct {
	c chan error
}

func newTarget(f func() error, deps ...Target) target {
	if f == nil {
		panic("nil func")
	}
	t := target{c: make(chan error, 1)}
	go t.run(f, deps...)
	return t
}

func (t *target) run(f func() error, deps ...Target) {
	for _, dep := range deps {
		if err := dep.Result(); err != nil {
			t.c <- err
			return
		}
	}
	t.c <- f()
}

func (t *target) Result() error {
	err := <-t.c
	t.c <- err
	return err
}

type ErrTarget struct {
	Err error
}

func (e ErrTarget) Result() error { return e.Err }

func (e ErrTarget) Pkgfile() string {
	panic(fmt.Sprintf("PkgFile called on ErrTarget: %v", e.Err))
}

func (e ErrTarget) Objfile() string {
	panic(fmt.Sprintf("Objfile called on ErrTarget: %v", e.Err))
}

// nilTarget always returns nil immediately.
type nilTarget struct{}

func (*nilTarget) Result() error { return nil }
