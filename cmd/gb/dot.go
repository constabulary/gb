package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/constabulary/gb"
)

func printActions(w io.Writer, a *gb.Action) {
	fmt.Fprintf(w, "digraph %q {\n", a.Name)
	seen := make(map[*gb.Action]bool)
	print0(w, seen, a)
	fmt.Fprintf(w, "}\n")
}

func print0(w io.Writer, seen map[*gb.Action]bool, a *gb.Action) {
	if seen[a] {
		return
	}

	split := func(s string) string {
		return strings.Replace(strings.Replace(s, ": ", "\n", -1), ",", "\n", -1)
	}

	for _, d := range a.Deps {
		print0(w, seen, d)
		fmt.Fprintf(w, "%q -> %q;\n", split(a.Name), split(d.Name))
	}

	seen[a] = true
}
