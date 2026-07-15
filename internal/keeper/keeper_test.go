package keeper

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// TestKeeperSetGetRoundTrip proves the split: write via the cnos CLI, read the
// raw value via the cnos Go client. Skipped where the cnos CLI is absent.
// Isolated via HOME/USERPROFILE so it never touches the real ~/.cnos.
func TestKeeperSetGetRoundTrip(t *testing.T) {
	if _, err := exec.LookPath("cnos"); err != nil {
		t.Skip("cnos CLI not installed (npm i -g @kitsy/cnos-cli)")
	}
	tmp := t.TempDir()
	t.Setenv("USERPROFILE", tmp) // Windows home for cnos + Go
	t.Setenv("HOME", tmp)        // unix home
	t.Setenv("HEY_HOME", filepath.Join(tmp, "hey"))
	t.Setenv(PassphraseEnv, "testpass123") // non-interactive vault auth

	if err := Set("gh-test", "ghp_secretVALUE"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := Get("gh-test")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "ghp_secretVALUE" {
		t.Fatalf("Get = %q, want ghp_secretVALUE", got)
	}
}
