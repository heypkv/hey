//go:build windows

package proc

import (
	"fmt"
	"os/exec"
	"strconv"
)

// TermSignal makes a best-effort graceful stop of pid. A detached console
// process on Windows cannot receive Unix-style signals, so we ask taskkill to
// terminate it *without* /F (which posts a close request that GUI and some
// console apps honor). Anything that ignores it is force-killed by the
// caller's KillTree fallback after the grace period, so a failure here is not
// fatal — packs that need a clean shutdown (e.g. postgres) define an explicit
// stop command instead.
func TermSignal(pid int, sig string) error {
	if out, err := exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T").CombinedOutput(); err != nil {
		return fmt.Errorf("taskkill pid %d: %v: %s", pid, err, out)
	}
	return nil
}
