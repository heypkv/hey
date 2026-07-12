//go:build !windows

package proc

import (
	"os/exec"
	"syscall"
)

// Detach configures cmd so the child survives hey's exit: a new session makes
// the child the leader of its own process group (which KillTree relies on).
func Detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
