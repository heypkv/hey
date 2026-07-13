package main

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestDoRunsSystemToolWithTemplatingAndCapture drives the plan executor against
// a mock system tool: templating, capture, and output flow, with --yes bypassing
// the sensitive-step consent gate. No network, no real tools.
func TestDoRunsSystemToolWithTemplatingAndCapture(t *testing.T) {
	t.Setenv("HEY_HOME", t.TempDir())

	// Build the mock "system tool" and put its dir first on PATH.
	binDir := t.TempDir()
	name := "heyplanprobe"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	build := exec.Command("go", "build", "-o", filepath.Join(binDir, name), "../../internal/planprobe")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build planprobe: %v: %s", err, out)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	planJSON := `{"hey_plan":1,"intent":"echo",
		"inputs":[{"name":"who","default":"world"}],
		"steps":[{"id":"e","tool":{"system":"heyplanprobe"},"sensitive":true,"run":["hi","{{ inputs.who }}"],"capture":"text"}],
		"output":"e"}`
	pf := filepath.Join(t.TempDir(), "echo.plan.json")
	if err := os.WriteFile(pf, []byte(planJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	// A local plan file is untrusted → needs --allow-untrusted; --yes bypasses
	// the sensitive-step consent.
	out := captureStdout(t, func() error {
		return cmdDo([]string{pf, "--yes", "--allow-untrusted", "--param", "who=earth"})
	})
	if !strings.Contains(out, "planprobe hi earth") {
		t.Fatalf("expected the tool's output to flow through, got %q", out)
	}
}

// TestDoUntrustedLocalPlanRefused: a local plan file without --allow-untrusted
// is refused before anything runs.
func TestDoUntrustedLocalPlanRefused(t *testing.T) {
	pf := filepath.Join(t.TempDir(), "x.plan.json")
	os.WriteFile(pf, []byte(`{"hey_plan":1,"intent":"x","steps":[{"id":"a","tool":{"system":"nmap"}}]}`), 0o644)
	err := cmdDo([]string{pf})
	if err == nil || !strings.Contains(err.Error(), "UNTRUSTED") {
		t.Fatalf("local plan without --allow-untrusted must be refused, got %v", err)
	}
}

func captureStdout(t *testing.T, f func() error) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	runErr := f()
	w.Close()
	os.Stdout = old
	data, _ := io.ReadAll(r)
	if runErr != nil {
		t.Fatalf("cmdDo: %v", runErr)
	}
	return string(data)
}
