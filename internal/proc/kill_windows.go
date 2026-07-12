//go:build windows

package proc

import (
	"fmt"
	"os/exec"
	"strconv"
)

// KillTree force-terminates pid and its whole child tree. Apps may have
// spawned helpers (e.g. headless Chrome for PDF), so tree-kill matters.
func KillTree(pid int) error {
	out, err := exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F").CombinedOutput()
	if err != nil {
		return fmt.Errorf("taskkill pid %d: %v: %s", pid, err, out)
	}
	return nil
}
