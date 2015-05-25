package cmd

import (
	"errors"
	"reflect"
	"testing"
)

func TestTestFlagsPreParse(t *testing.T) {
	tests := []struct {
		args  []string // The command line arguments to parse
		pargs []string // The expected arguments for flag.Parse
		eargs []string // The expected "extra" arguments to pass to the test binary
		err   error
	}{
		{
			args:  []string{"-q", "-test", "-debug"},
			pargs: []string{"-q"},
			eargs: []string{"-q", "-test", "-debug"},
		}, {
			args:  []string{"-v", "-debug", "package_name"},
			pargs: []string{"-v", "package_name"},
			eargs: []string{"-v", "-debug"},
		}, {
			args:  []string{"-q", "-debug", "package_name"},
			pargs: []string{"-q", "package_name"},
			eargs: []string{"-q", "-debug"},
		}, {
			args:  []string{"-bench"},
			eargs: []string{"-bench"},
		}, {
			args:  []string{"-bench=."},
			eargs: []string{"-bench=."},
		}, {
			args:  []string{"-bench", "."},
			eargs: []string{"-bench=."},
		}, {
			args:  []string{"-bench", "Test*"},
			eargs: []string{"-bench=Test*"},
		}, {
			args:  []string{"-benchmem"},
			eargs: []string{"-benchmem"},
		}, {
			args:  []string{"-benchtime", "2s"},
			eargs: []string{"-benchtime=2s"},
		}, {
			args:  []string{"-blockprofile", "profile"},
			eargs: []string{"-blockprofile=profile"},
		}, {
			args:  []string{"-blockprofile", "profile", "-cover"},
			pargs: []string{"-cover"},
			eargs: []string{"-blockprofile=profile"},
		}, {
			args:  []string{"-blockprofilerate", "2"},
			eargs: []string{"-blockprofilerate=2"},
		}, {
			args:  []string{"-coverprofile", "c.out"},
			eargs: []string{"-coverprofile=c.out"},
		}, {
			args:  []string{"-cpu", "1"},
			eargs: []string{"-cpu=1"},
		}, {
			args:  []string{"-cpu", "1"},
			eargs: []string{"-cpu=1"},
		}, {
			args:  []string{"-timeout"},
			eargs: []string{"-timeout"},
		}, {
			args:  []string{"-timeout", "2s"},
			eargs: []string{"-timeout=2s"},
		}, {
			args:  []string{"-test.run", "test"},
			eargs: []string{"-test.run=test"},
		}, {
			args:  []string{"-test.bench", "Test*"},
			eargs: []string{"-test.bench=Test*"},
		}, {
			args:  []string{"-test.bench=Test*"},
			eargs: []string{"-test.bench=Test*"},
		}, {
			args: []string{"-test.run", "Test*", "-test.run", "Test2*"},
			err:  errors.New("\"run\" flag may be set only once"),
		}, {
			args:  []string{"-cover=true"},
			pargs: []string{"-cover=true"},
		}, {
			args:  []string{"-cover=false"},
			pargs: []string{"-cover=false"},
		}, {
			args: []string{"-cover=notabool"},
			err:  errors.New("illegal bool flag value notabool"),
		}, {
			args: []string{"-run", "Test*", "-run", "Test2*"},
			err:  errors.New("\"run\" flag may be set only once"),
		}, {
			args:  []string{"-short"},
			eargs: []string{"-short"},
		}, {
			args:  []string{"-memprofilerate", "1"},
			eargs: []string{"-memprofilerate=1"},
		}, {
			args:  []string{"-coverpkg", "package"},
			pargs: []string{"-coverpkg", "package"},
		}}

	for _, tt := range tests {
		for k, v := range testFlagDefn {
			if v.present {
				testFlagDefn[k].present = false
			}
		}
		pargs, eargs, err := TestFlagsExtraParse(tt.args)
		if tt.err != nil && (err == nil || (err != nil && tt.err.Error() != err.Error())) {
			t.Errorf("TestExtraFlags(%v): want err = '%v', got = '%v'", tt.args, tt.err, err)
		} else if tt.err == nil && (!reflect.DeepEqual(pargs, tt.pargs) || !reflect.DeepEqual(eargs, tt.eargs)) {
			t.Errorf("TestExtraFlags(%v): want (%v,%v), got (%v,%v)", tt.args, tt.pargs, tt.eargs, pargs, eargs)
		}
	}
}

func TestTestFlags(t *testing.T) {
	tests := []struct {
		eargs []string // Extra test binary arguments
		targs []string // The expected test binary arguments
	}{
		{
			eargs: []string{"-q", "-debug"},
			targs: []string{"-test.v", "-debug"},
		}, {
			eargs: []string{"-v", "-debug"},
			targs: []string{"-test.v", "-debug"},
		}, {
			eargs: []string{"-bench"},
			targs: []string{"-test.bench"},
		}, {
			eargs: []string{"-bench", "."},
			targs: []string{"-test.bench", "."},
		}, {
			eargs: []string{"-bench='Test*'"},
			targs: []string{"-test.bench='Test*'"},
		}, {
			eargs: []string{"-benchmem"},
			targs: []string{"-test.benchmem"},
		}, {
			eargs: []string{"-benchtime"},
			targs: []string{"-test.benchtime"},
		}, {
			eargs: []string{"-benchtime", "2s"},
			targs: []string{"-test.benchtime", "2s"},
		}, {
			eargs: []string{"-benchtime=2s"},
			targs: []string{"-test.benchtime=2s"},
		}, {
			eargs: []string{"-blockprofile", "profile"},
			targs: []string{"-test.blockprofile", "profile"},
		}, {
			eargs: []string{"-blockprofile=profile"},
			targs: []string{"-test.blockprofile=profile"},
		}, {
			eargs: []string{"-blockprofile"},
			targs: []string{"-test.blockprofile"},
		}, {
			eargs: []string{"-cpuprofile"},
			targs: []string{"-test.cpuprofile"},
		}, {
			eargs: []string{"-memprofile"},
			targs: []string{"-test.memprofile"},
		}, {
			eargs: []string{"-short"},
			targs: []string{"-test.short"},
		}, {
			eargs: []string{"-memprofilerate", "1"},
			targs: []string{"-test.memprofilerate", "1"},
		}, {
			eargs: []string{"-test.run=test"},
			targs: []string{"-test.run=test"},
		}, {
			eargs: []string{"-test.short"},
			targs: []string{"-test.short"},
		}}

	for _, tt := range tests {
		targs := TestFlags(tt.eargs)
		if !reflect.DeepEqual(targs, tt.targs) {
			t.Errorf("TestFlags(%v): want %v, got %v", tt.eargs, tt.targs, targs)
		}
	}
}
