//go:build !windows

package proc

import (
	"syscall"
	"time"
)

// KillTree terminates the process group led by pid (Detach used Setsid, so
// the child is its group's leader): SIGTERM, 5s grace, then SIGKILL.
func KillTree(pid int) error {
	if err := syscall.Kill(-pid, syscall.SIGTERM); err != nil {
		// Group may already be gone; try the single process before giving up.
		if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
			return nil // already dead
		}
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !Alive(pid) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	_ = syscall.Kill(-pid, syscall.SIGKILL)
	_ = syscall.Kill(pid, syscall.SIGKILL)
	return nil
}
