package pprof

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"runtime/pprof"
	"time"

	log "github.com/sirupsen/logrus"
)

// dump writes the heap and goroutine trace to |w|.
func dump(w io.Writer) {
	pprof.Lookup("heap").WriteTo(w, 1)
	pprof.Lookup("goroutine").WriteTo(w, 1)
}

// TODO is this really worth extracting? Does it work correctly writing to a file in this way? seems like no
// toggleProfiler starts and stops a long-running CPU profile using pprof. The
// profile is written to /var/tmp/profile_${PID}_${TIMESTAMP}.pprof where
// TIMESTAMP represents the epoch time when the profiling session began.
func toggleProfiler(w io.Writer, f *os.File) {
	if w == nil {
		var err error

		filename := fmt.Sprintf("/var/tmp/profile_%d_%d.pprof",
			os.Getpid(), time.Now().Unix())

		f, err = os.Create(filename)
		if err == nil {
			w = bufio.NewWriter(f)
			pprof.StartCPUProfile(w)
		} else {
			log.WithField("err", err).Error("could not begin CPU profiling")
		}
	} else {
		pprof.StopCPUProfile()
		f.Close()
		w = nil
		f = nil
	}
}
