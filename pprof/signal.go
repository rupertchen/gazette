package pprof

import (
	"io"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
)

var (
	traceEnabled     bool
	previousLogLevel log.Level
)

// RegisterSignalHandlers registers signal handlers for debugging and
// profiling.
func RegisterSignalHandlers() {
	notifyChan := make(chan os.Signal, 1)
	signal.Notify(notifyChan, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)
	go func() {
		var profileWriter io.Writer
		var profileFile *os.File

		for {
			switch <-notifyChan {
			case syscall.SIGQUIT:
				dump(os.Stdout)
			case syscall.SIGUSR1:
				toggleProfiler(profileWriter, profileFile)
			case syscall.SIGUSR2:
				toggleTrace()
			}
		}
	}()
}

// toggleTrace toggles debug logging
func toggleTrace() {
	if traceEnabled {
		log.SetLevel(previousLogLevel)
	} else {
		previousLogLevel = log.GetLevel()
		log.SetLevel(log.DebugLevel)
	}
	traceEnabled = !traceEnabled
}
