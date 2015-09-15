package log

import (
	"fmt"
	"os"
)

var (
	// Quiet suppresses all logging output below ERROR
	Quiet bool

	// Verbose enables logging output below INFO
	Verbose bool
)

func Fatalf(format string, args ...interface{}) {
	fmt.Printf("FATAL "+format+"\n", args...)
	os.Exit(1)
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
