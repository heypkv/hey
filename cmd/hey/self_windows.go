//go:build windows

package main

import (
	"fmt"
	"os"
)

// replaceSelf overwrites the running binary. Windows won't let you delete or
// overwrite a running .exe, but it will let you RENAME it — so move the running
// exe aside to <exe>.old and write the new binary at the original path. The
// stale .old is cleaned up on the next update.
func replaceSelf(self, newBin string) error {
	old := self + ".old"
	_ = os.Remove(old) // clear a leftover from a previous update
	if err := os.Rename(self, old); err != nil {
		return fmt.Errorf("move the running exe aside (a --system install may need an elevated shell): %w", err)
	}
	if err := copyFile(newBin, self, 0o755); err != nil {
		// Roll back so hey isn't left missing.
		_ = os.Rename(old, self)
		return err
	}
	return nil
}
