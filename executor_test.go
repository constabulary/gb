package gb

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"testing"
)

func TestExecuteBuildAction(t *testing.T) {
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
	}, {
		pkg: "x",
		err: errors.New("import cycle detected: x -> y -> x"),
	}, {
		pkg: "h", // imports "blank", which is blank, see issue #131
		err: fmt.Errorf("no buildable Go source files in %s", filepath.Join(getwd(t), "testdata", "src", "blank")),
	}}

	for _, tt := range tests {
		ctx := testContext(t)
		pkg, err := ctx.ResolvePackage(tt.pkg)
		if !sameErr(err, tt.err) {
			t.Errorf("ctx.ResolvePackage(%v): want %v, got %v", tt.pkg, tt.err, err)
			continue
		}
		if err != nil {
			continue
		}
		action, err := BuildPackages(pkg)
		if err != nil {
			t.Errorf("BuildAction(%v): ", tt.pkg, err)
			continue
		}
		if err := Execute(action); !sameErr(err, tt.err) {
			t.Errorf("Execute(%v): want: %v, got %v", action.Name, tt.err, err)
		}
		ctx.Destroy()
	}
}

var niltask = TaskFn(func() error { return nil })

var executorTests = []struct {
	action *Action // root action
	err    error   // expected error
}{{
	action: &Action{
		Name: "no error",
		Task: niltask,
	},
}, {
	action: &Action{
		Name: "root error",
		Task: TaskFn(func() error {
			return io.EOF
		}),
	},
	err: io.EOF,
}, {
	action: &Action{
		Name: "child, child, error",
		Task: TaskFn(func() error {
			return fmt.Errorf("I should not have been called")
		}),
		Deps: []*Action{&Action{
			Name: "child, error",
			Task: niltask,
			Deps: []*Action{&Action{
				Name: "error",
				Task: TaskFn(func() error {
					return io.EOF
				}),
			}},
		}},
	},
	err: io.EOF,
}, {
	action: &Action{
		Name: "once only",
		Task: TaskFn(func() error {
			if c1 != 1 || c2 != 1 || c3 != 1 {
				return fmt.Errorf("unexpected count, c1: %v, c2: %v, c3: %v", c1, c2, c3)
			}
			return nil
		}),
		Deps: []*Action{createDag()},
	},
}, {
	action: &Action{
		Name: "failure count",
		Task: TaskFn(func() error {
			return fmt.Errorf("I should not have been called")
		}),
		Deps: []*Action{createFailDag()},
	},
	err: fmt.Errorf("task3 called 1 time"),
}}

func createDag() *Action {
	task1 := TaskFn(func() error { c1++; return nil })
	task2 := TaskFn(func() error { c2++; return nil })
	task3 := TaskFn(func() error { c3++; return nil })

	action1 := Action{Name: "c1", Task: task1}
	action2 := Action{Name: "c2", Task: task2}
	action3 := Action{Name: "c3", Task: task3}

	action1.Deps = append(action1.Deps, &action2, &action3)
	action2.Deps = append(action2.Deps, &action3)
	return &action1
}

func createFailDag() *Action {
	task1 := TaskFn(func() error { c1++; return nil })
	task2 := TaskFn(func() error { c2++; return fmt.Errorf("task2 called %v time", c2) })
	task3 := TaskFn(func() error { c3++; return fmt.Errorf("task3 called %v time", c3) })

	action1 := Action{Name: "c1", Task: task1}
	action2 := Action{Name: "c2", Task: task2}
	action3 := Action{Name: "c3", Task: task3}

	action1.Deps = append(action1.Deps, &action2, &action3)
	action2.Deps = append(action2.Deps, &action3)
	return &action1
}

var c1, c2, c3 int

func executeReset() {
	c1 = 0
	c2 = 0
	c3 = 0
	// reset executor test variables
}

func TestExecute(t *testing.T) {
	for _, tt := range executorTests {
		executeReset()
		got := Execute(tt.action)
		if !reflect.DeepEqual(got, tt.err) {
			t.Errorf("Execute: %v: want err: %v, got err %v", tt.action.Name, tt.err, got)
		}
	}
}

func testExecuteConcurrentN(t *testing.T, n int) {
	for _, tt := range executorTests {
		executeReset()
		got := ExecuteConcurrent(tt.action, n)
		if !reflect.DeepEqual(got, tt.err) {
			t.Errorf("ExecuteConcurrent(%v): %v: want err: %v, got err %v", n, tt.action.Name, tt.err, got)
		}
	}
}

func TestExecuteConcurrent1(t *testing.T) { testExecuteConcurrentN(t, 1) }
func TestExecuteConcurrent2(t *testing.T) { testExecuteConcurrentN(t, 2) }
func TestExecuteConcurrent4(t *testing.T) { testExecuteConcurrentN(t, 4) }
func TestExecuteConcurrent7(t *testing.T) { testExecuteConcurrentN(t, 7) }
