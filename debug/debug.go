// debug provides a light weight debug facility.
// Usage is via the DEBUG environment variable. Any non empty value will
// enable debug logging. For example
//
//     DEBUG=. gb
//
// The period is a hint that the value passed to DEBUG is a regex, which matches
// files, or packages present in the file part of the file/line pair logged as a
// prefix of the log line. (not implemented yet)
//
// Debug output is send to os.Stderr, there is no facility to change this.
package debug

import (
	"fmt"
	"log"
	"os"
)

var debug = os.Getenv("DEBUG")

var logger = log.New(os.Stderr, "", log.Ldate|log.Ltime|log.Lshortfile)

func Debugf(format string, args ...interface{}) {
	if len(debug) == 0 {
		return
	}
	logger.Output(2, fmt.Sprintf(format, args...))
}
