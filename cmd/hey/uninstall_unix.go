//go:build !windows

package main

import (
	"fmt"
	"os"
)

// removeSelf deletes the running hey binary. On unix the file backing a
// running process can be unlinked immediately (the inode survives until the
// process exits), so a plain remove works. The installer does not edit shell
// profiles, so there is no PATH entry to clean up.
func removeSelf(path string) error {
	if err := os.Remove(path); err != nil {
		return err
	}
	fmt.Println("(if you added hey's directory to your shell PATH by hand, remove that line too)")
	return nil
}
