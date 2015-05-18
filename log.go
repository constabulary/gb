package gb

import (
	"fmt"
	"os"
)

var (
	// Quiet suppresses all logging output below ERROR
	Quiet = false

	// Verbose enables logging output below INFO
	Verbose = false
)

func Fatalf(format string, args ...interface{}) {
	fmt.Printf("FATAL "+format+"\n", args...)
	os.Exit(1)
}

func Errorf(format string, args ...interface{}) {
	fmt.Printf("ERROR "+format+"\n", args...)
}

func Warnf(format string, args ...interface{}) {
	fmt.Printf("WARNING "+format+"\n", args...)
}

func Infof(format string, args ...interface{}) {
	if !Quiet {
		if Verbose {
			fmt.Printf("INFO "+format+"\n", args...)
		} else {
			fmt.Printf(format+"\n", args...)
		}
	}
}

func Debugf(format string, args ...interface{}) {
	if Verbose && !Quiet {
		fmt.Printf("DEBUG "+format+"\n", args...)
	}
}
