package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// testFlagSpec defines a flag we know about.
type testFlagSpec struct {
	boolVar    bool // True if the flag is type bool
	passToTest bool // pass to Test
	passToAll  bool // pass to test plugin and test binary
	present    bool // The flag has been seen
}

// testFlagDefn is the set of flags we process.
var testFlagDefn = map[string]*testFlagSpec{
	// local to the test plugin
	"q":         {boolVar: true, passToAll: true},
	"v":         {boolVar: true, passToAll: true},
	"cover":     {boolVar: true},
	"coverpkg":  {},
	"covermode": {},

	// Passed to the test binary
	"bench":            {passToTest: true},
	"benchmem":         {boolVar: true, passToTest: true},
	"benchtime":        {passToTest: true},
	"coverprofile":     {passToTest: true},
	"cpu":              {passToTest: true},
	"cpuprofile":       {passToTest: true},
	"memprofile":       {passToTest: true},
	"memprofilerate":   {passToTest: true},
	"blockprofile":     {passToTest: true},
	"blockprofilerate": {passToTest: true},
	"outputdir":        {passToTest: true},
	"parallel":         {passToTest: true},
	"run":              {passToTest: true},
	"short":            {boolVar: true, passToTest: true},
	"timeout":          {passToTest: true},
}

// TestFlags appends "-test." for flags that are passed to the test binary.
func TestFlags(testArgs []string) []string {
	var targs []string
	for _, arg := range testArgs {
		var nArg, nVal, fArg string
		fArg = arg
		if !strings.Contains(arg, "-test.") {
			nArg = strings.TrimPrefix(arg, "-")
			if strings.Contains(nArg, "=") {
				nArgVal := strings.Split(nArg, "=")
				nArg, nVal = nArgVal[0], nArgVal[1]
			}
			if val, ok := testFlagDefn[nArg]; ok {
				// Special handling for -q, needs to be -test.v when passed to the test
				if nArg == "q" {
					nArg = "v"
				}
				if val.passToTest || val.passToAll {
					fArg = "-test." + nArg
				}
			}
			if nVal != "" {
				fArg = fArg + "=" + nVal
			}
		}
		targs = append(targs, fArg)
	}
	return targs
}

// TestFlagsExtraParse is used to separate known arguments from unknown
// arguments passed on the command line. Returns a string slice of test plugin
// arguments (parseArgs), and a slice of string arguments for the test binary
// (extraArgs). An error is returned if an argument is used twice, or an
// argument value is incorrect.
func TestFlagsExtraParse(args []string) (parseArgs []string, extraArgs []string, err error) {
	argsLen := len(args)

	for x := 0; x < argsLen; x++ {
		nArg := args[x]
		val, ok := testFlagDefn[strings.TrimPrefix(nArg, "-")]
		if !strings.HasPrefix(nArg, "-") || (ok && !val.passToTest) {
			err = setArgFound(nArg)
			if err != nil {
				return
			}
			parseArgs = append(parseArgs, nArg)
			if ok && val.passToAll {
				extraArgs = append(extraArgs, nArg)
			}
			continue
		}

		var hadTestPrefix bool
		hasEqual := strings.Contains(nArg, "=")
		if !hasEqual && (x+1 < argsLen && !strings.HasPrefix(args[x+1], "-")) {
			if strings.Contains(nArg, "-test.") {
				hadTestPrefix = true
				nArg = strings.TrimPrefix(nArg, "-test.")
			} else {
				nArg = strings.TrimPrefix(nArg, "-")
			}
			err = setArgFound(nArg)
			if err != nil {
				return
			}
			// Check the spec for arguments that consume the next argument
			if val, ok := testFlagDefn[nArg]; ok {
				if !val.boolVar {
					nArg = nArg + "=" + args[x+1]
					x++
				}
			}
		} else if hasEqual {
			// The argument has an embedded value, here we can do some basic
			// checking.
			sArgs := strings.Split(nArg, "=")
			tArg, tVal := strings.TrimPrefix(sArgs[0], "-"), sArgs[1]
			if val, ok := testFlagDefn[tArg]; ok {
				if val.boolVar {
					if err = checkBoolFlag(tVal); err != nil {
						return
					}
				}
				if !val.passToTest {
					parseArgs = append(parseArgs, nArg)
					continue
				}
			}
		}

		// Append "-" to the argument, and "-test." if "-test." was previously
		// trimmed.
		if nArg[0] != '-' {
			pre := "-"
			if hadTestPrefix {
				pre = "-test."
			}
			nArg = pre + nArg
		}
		extraArgs = append(extraArgs, nArg)
	}

	return
}

// setArgFound checks the argument spec to see if arg has already been
// encountered. If it has, then an error is returned.
func setArgFound(arg string) error {
	var err error
	nArg := strings.TrimPrefix(arg, "-")
	if val, ok := testFlagDefn[nArg]; ok {
		if val.present {
			err = fmt.Errorf("%q flag may be set only once", arg)
		} else {
			testFlagDefn[nArg].present = true
		}
	}
	return err
}

// checkBoolFlag checks the value to ensure it is a boolean, if not an error is
// returned.
func checkBoolFlag(value string) error {
	var nErr error
	_, err := strconv.ParseBool(value)
	if err != nil {
		nErr = errors.New("illegal bool flag value " + value)
	}
	return nErr
}
