//go:build !windows

package proc

import "syscall"

// TermSignal delivers a graceful termination signal to pid. sig is one of
// "term" (default), "int" or "kill". A service's own stop command is preferred
// when the pack defines one; this is the signal-based fallback path.
func TermSignal(pid int, sig string) error {
	s := syscall.SIGTERM
	switch sig {
	case "int":
		s = syscall.SIGINT
	case "kill":
		s = syscall.SIGKILL
	}
	return syscall.Kill(pid, s)
}
