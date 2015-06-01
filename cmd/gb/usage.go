package main

import (
	"bufio"
	"io"
	"os"
	"sort"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"

	"github.com/constabulary/gb/cmd"
)

var usageTemplate = `gb, a project based build tool for the Go programming language.

Usage:

        gb command [arguments]

Valid commands are:{{range .}}
        gb {{.Name | printf "%-11s"}} - {{.ShortDesc}}{{end}}
`

// tmpl executes the given template text on data, writing the result to w.
func tmpl(w io.Writer, text string, data interface{}) {
	t := template.New("top")
	t.Funcs(template.FuncMap{"trim": strings.TrimSpace, "capitalize": capitalize})
	template.Must(t.Parse(text))
	if err := t.Execute(w, data); err != nil {
		panic(err)
	}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToTitle(r)) + s[n:]
}

func printUsage(w io.Writer) {
	var sortedKeys []string
	for k := range commands {
		sortedKeys = append(sortedKeys, k)
	}

	sort.Strings(sortedKeys)
	var cmds []*cmd.Command
	for _, c := range sortedKeys {
		cmds = append(cmds, commands[c])
	}
	bw := bufio.NewWriter(w)
	tmpl(bw, usageTemplate, cmds)
	bw.Flush()
}

func usage() {
	printUsage(os.Stderr)
	os.Exit(2)
}
