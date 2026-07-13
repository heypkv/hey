// deployprobe is a synthetic "window" application used only by the deploy
// install/run tests. Like internal/testapp and internal/svcprobe it is never
// shipped. It writes a marker file (its first argument) to prove hey launched
// it, then exits 0 — enough to drive install → extract → launch → cleanup for
// the `interface: window` bundle path with no network and no real GUI.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "deployprobe: expected a marker path argument")
		os.Exit(1)
	}
	if err := os.WriteFile(os.Args[1], []byte("launched\n"), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "deployprobe:", err)
		os.Exit(1)
	}
}
