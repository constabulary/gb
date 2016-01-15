package gb

import (
	"reflect"
	"sort"
	"testing"
)

func TestBuildAction(t *testing.T) {
	var tests = []struct {
		pkg    string
		action *Action
		err    error
	}{{
		pkg: "a",
		action: &Action{
			Name: "build: a",
			Deps: []*Action{&Action{Name: "compile: a"}},
		},
	}, {
		pkg: "b",
		action: &Action{
			Name: "build: b",
			Deps: []*Action{
				&Action{
					Name: "link: b",
					Deps: []*Action{
						&Action{
							Name: "compile: b",
							Deps: []*Action{
								&Action{
									Name: "compile: a",
								}},
						},
					}},
			},
		},
	}, {
		pkg: "c",
		action: &Action{
			Name: "build: c",
			Deps: []*Action{
				&Action{
					Name: "compile: c",
					Deps: []*Action{
						&Action{
							Name: "compile: a",
						}, &Action{
							Name: "compile: d.v1",
						}},
				}},
		},
	}}
	for _, tt := range tests {
		ctx := testContext(t)
		defer ctx.Destroy()
		pkg, err := ctx.ResolvePackage(tt.pkg)
		if !reflect.DeepEqual(err, tt.err) {
			t.Errorf("ctx.ResolvePackage(%v): want %v, got %v", tt.pkg, tt.err, err)
			continue
		}
		if err != nil {
			continue
		}
		got, err := BuildPackages(pkg)
		if !reflect.DeepEqual(err, tt.err) {
			t.Errorf("BuildAction(%v): want %v, got %v", tt.pkg, tt.err, err)
			continue
		}
		deleteTasks(got)

		if !reflect.DeepEqual(tt.action, got) {
			t.Errorf("BuildAction(%v): want %#+v, got %#+v", tt.pkg, tt.action, got)
		}

		// double underpants
		sameAction(t, got, tt.action)
	}
}

func sameAction(t *testing.T, want, got *Action) {
	if want.Name != got.Name {
		t.Errorf("sameAction: names do not match, want: %v, got %v", want.Name, got.Name)
		return
	}
	if len(want.Deps) != len(got.Deps) {
		t.Errorf("sameAction(%v, %v): deps: len(want): %v, len(got): %v", want.Name, got.Name, len(want.Deps), len(got.Deps))
		return
	}
	w, g := make(map[string]*Action), make(map[string]*Action)
	for _, a := range want.Deps {
		w[a.Name] = a
	}
	for _, a := range got.Deps {
		g[a.Name] = a
	}
	var wk []string
	for k := range w {
		wk = append(wk, k)
	}
	sort.Strings(wk)
	for _, a := range wk {
		g, ok := g[a]
		if !ok {
			t.Errorf("sameAction(%v, %v): deps: want %v, got nil", want.Name, got.Name, a)
			continue
		}
		sameAction(t, w[a], g)
	}
}

func deleteTasks(a *Action) {
	for _, d := range a.Deps {
		deleteTasks(d)
	}
	a.Run = nil
}
