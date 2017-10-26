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

func RegisterSignalHandlers() {
	// TODO (rupert): break out the actual functionality from the signal handling?
	// Signal handlers for debugging and profiling:
	// SIGQUIT: Dump a one-time heap and goroutine trace to stdout.
	// SIGUSR1: Start a long-running CPU profile using pprof. Write the profile
	// to this path: /var/tmp/profile_${PID}_${TIMESTAMP}.pprof where TIMESTAMP
	// represents the epoch time when the profiling session began.
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
