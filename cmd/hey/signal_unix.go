//go:build !windows

package main

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

// forwardSignals relays SIGINT/SIGTERM to the foreground child so Ctrl+C
// reaches it even when the shell delivers the signal to hey. Returns a stop
// function for cleanup.
func forwardSignals(cmd *exec.Cmd) func() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		for s := range ch {
			if cmd.Process != nil {
				_ = cmd.Process.Signal(s)
			}
		}
	}()
	return func() {
		signal.Stop(ch)
		close(ch)
	}
}
