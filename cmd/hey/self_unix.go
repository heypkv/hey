//go:build !windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// replaceSelf overwrites the running binary. On unix a running executable's
// file can be replaced via rename in the same directory (the process keeps its
// open text image until exit), so write the new binary beside it and rename
// over it atomically.
func replaceSelf(self, newBin string) error {
	dir := filepath.Dir(self)
	staged := filepath.Join(dir, ".hey.update")
	if err := copyFile(newBin, staged, 0o755); err != nil {
		return fmt.Errorf("stage new binary in %s (need write permission — a --system install may need sudo): %w", dir, err)
	}
	if err := os.Rename(staged, self); err != nil {
		os.Remove(staged)
		return err
	}
	return nil
}
