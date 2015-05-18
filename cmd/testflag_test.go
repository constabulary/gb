package cmd

import (
	"flag"
	"reflect"
	"testing"
)

func TestTestExtraFlags(t *testing.T) {
	tests := []struct {
		args    []string // The command line arguments to parse
		pargs   []string // The expected arguments for flag.Parse
		eargs   []string // The expected "extra" arguments to pass to the test binary
		flagSet func() *flag.FlagSet
		err     error
	}{
		{
			args:  []string{"-q", "-test", "-debug"},
			pargs: []string{"-q", "-test"},
			eargs: []string{"-debug"},
			flagSet: func() *flag.FlagSet {
				var test, q bool
				fs := flag.NewFlagSet("test", flag.ExitOnError)
				fs.BoolVar(&q, "q", false, "quiet")
				fs.BoolVar(&test, "test", false, "test bool")
				return fs
			},
		}, {
			args:  []string{"-q", "-debug", "package_name"},
			pargs: []string{"-q", "package_name"},
			eargs: []string{"-debug"},
			flagSet: func() *flag.FlagSet {
				var q bool
				fs := flag.NewFlagSet("test", flag.ExitOnError)
				fs.BoolVar(&q, "q", false, "quiet")
				return fs
			},
		}}

	for _, tt := range tests {
		pargs, eargs := TestExtraFlags(tt.flagSet(), tt.args)
		if !reflect.DeepEqual(pargs, tt.pargs) || !reflect.DeepEqual(eargs, tt.eargs) {
			t.Errorf("TestExtraFlags(%v): want (%v,%v), got (%v,%v)",
				tt.args, tt.pargs, tt.eargs, pargs, eargs)
		}
	}
}

func TestTestFlags(t *testing.T) {
	tests := []struct {
		args    []string // The command line arguments to parse
		eargs   []string // Extra test binary arguments
		targs   []string // The expected test binary arguments
		flagSet func() *flag.FlagSet
		err     error
	}{
		{
			args:  []string{"-q"},
			eargs: []string{"-debug"},
			targs: []string{"-test.v", "-debug"},
			flagSet: func() *flag.FlagSet {
				var q bool
				fs := flag.NewFlagSet("test", flag.ExitOnError)
				fs.BoolVar(&q, "q", false, "quiet")
				return fs
			},
		}}

	for _, tt := range tests {
		fs := tt.flagSet()
		fs.Parse(tt.args)
		targs := TestFlags(fs, tt.eargs)
		if !reflect.DeepEqual(targs, tt.targs) {
			t.Errorf("TestFlags(%v): want %v, got %v", tt.args, tt.targs, targs)
		}
	}
}
