// Package browser opens URLs in the user's default browser.
package browser

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Open launches the default browser at url. Callers should treat failure as
// a warning, never an error — the URL is always printed anyway.
func Open(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// rundll32 avoids cmd.exe quoting pitfalls with & and ? in URLs.
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	go cmd.Wait() // reap; we don't care about the outcome
	return nil
}
