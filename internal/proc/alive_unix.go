//go:build !windows

package proc

import "syscall"

// Alive reports whether pid refers to a running process (signal 0 probe).
func Alive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}
