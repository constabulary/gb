package cmd

import (
	"flag"
	"strings"
)

// TestFlags appends "-test." to flags used with the test plugin to be passed
// to the test binary.
func TestFlags(flags *flag.FlagSet, testArgs []string) []string {
	var targs []string
	visitor := func(flag *flag.Flag) {
		if flag.Name == "q" || flag.Name == "v" {
			targs = append(targs, "-test.v")
		}
	}
	flags.Visit(visitor)
	targs = append(targs, testArgs...)
	return targs
}

// TestExtraFlags is used to separate known arguments from unknown arguments
// passed on the command line. Returns a string slice of known arguments
// (parseArgs), and a slice of string arguments for the test binary
// (extraArgs).
func TestExtraFlags(flags *flag.FlagSet, args []string) (parseArgs []string, extraArgs []string) {
	vargs := make(map[string]bool)
	eargs := make(map[string]bool)
	keysToSlice := func(m map[string]bool) []string {
		var s []string
		for k := range m {
			s = append(s, k)
		}
		return s
	}
	visitor := func(flag *flag.Flag) {
		for _, x := range args {
			arg := x
			if strings.HasPrefix(x, "-") {
				arg = strings.TrimPrefix(x, "-")
			}
			if flag.Name == arg {
				vargs[x] = true
				break
			}
		}
	}
	flags.VisitAll(visitor)
	for _, x := range args {
		if !strings.HasPrefix(x, "-") {
			vargs[x] = true
			continue
		}
		if _, ok := vargs[x]; !ok {
			eargs[x] = true
		}
	}
	return keysToSlice(vargs), keysToSlice(eargs)
}
