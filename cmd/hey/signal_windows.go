//go:build windows

package main

import (
	"os"
	"os/exec"
	"os/signal"
)

// forwardSignals: on Windows the foreground child shares hey's console, so
// Ctrl+C is delivered to the whole console process group by the OS. hey just
// ignores the interrupt itself so it survives long enough to report the
// child's exit code. Returns a stop function for cleanup.
func forwardSignals(cmd *exec.Cmd) func() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		for range ch {
			// swallow; the child receives its own CTRL_C_EVENT
		}
	}()
	return func() {
		signal.Stop(ch)
		close(ch)
	}
}
