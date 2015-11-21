package gb

import (
	"os"
	"os/signal"
	"sync"
)

// If we catch signal then this will get closed
var interrupted = make(chan struct{})

func processSignal() {
	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)
	go func() {
		<-sig
		close(interrupted)
	}()
}

// We want to process signal only once
var processSignalOnce sync.Once

func StartSignalHandler() {
	processSignalOnce.Do(processSignal)
}