// Package gogo/log controls the logging output for gb derived programs.
package gb

import (
	"log"
	"os"
)

var (
	// Quiet suppresses all logging output below ERROR
	Quiet = false

	// Verbose enables logging output below INFO
	Verbose = false

	// Logger is the log.Logger object that backs this logger.
	Logger = log.New(os.Stdout, "", log.LstdFlags)
)

var Fatalf = log.Fatalf

func Errorf(format string, args ...interface{}) {
	Logger.Printf("ERROR "+format, args...)
}

func Warnf(format string, args ...interface{}) {
	Logger.Printf("WARNING "+format, args...)
}

func Infof(format string, args ...interface{}) {
	if !Quiet {
		Logger.Printf("INFO "+format, args...)
	}
}

func Debugf(format string, args ...interface{}) {
	if Verbose && !Quiet {
		Logger.Printf("DEBUG "+format, args...)
	}
}
