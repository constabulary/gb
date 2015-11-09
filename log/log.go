package log

import "fmt"

var (
	// Verbose enables Debug logging
	Verbose bool
)

func Debugf(format string, args ...interface{}) {
	if Verbose {
		fmt.Printf("DEBUG "+format+"\n", args...)
	}
}
