//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kitsyai/hey/internal/proc"
)

// removeSelf handles Windows' inability to delete a running .exe: it strips
// hey's directory from the user PATH (the installer added it) and spawns a
// detached batch that waits for this process to exit, deletes the exe, removes
// the dedicated hey directory, then deletes itself.
func removeSelf(path string) error {
	dir := filepath.Dir(path)
	removeFromUserPath(dir)

	rmdir := ""
	if strings.EqualFold(filepath.Base(dir), "hey") {
		rmdir = `rmdir /q "` + dir + `" >nul 2>&1`
	}
	// %%~f0 (escaped for fmt) is the batch's own path, so it self-deletes last.
	script := fmt.Sprintf(`@echo off
:retry
del /f /q "%s" >nul 2>&1
if exist "%s" ( timeout /t 1 /nobreak >nul & goto retry )
%s
del /f /q "%%~f0" >nul 2>&1
`, path, path, rmdir)

	batch := filepath.Join(os.TempDir(), "hey-uninstall.cmd")
	if err := os.WriteFile(batch, []byte(script), 0o644); err != nil {
		return err
	}
	cmd := exec.Command("cmd", "/c", batch)
	proc.Detach(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	fmt.Println("(the binary is deleted the moment hey exits)")
	return nil
}

// removeFromUserPath drops dir from the user PATH via PowerShell (stdlib-only,
// no registry dependency). It rewrites PATH only when dir is actually present
// and removes just that exact entry — other entries (including empties) are
// left untouched, so it's a true no-op when hey wasn't on PATH. Best-effort.
func removeFromUserPath(dir string) {
	ps := "$d='" + dir + "'; " +
		"$p=[Environment]::GetEnvironmentVariable('Path','User'); " +
		"$parts=@($p -split ';'); " +
		"if ($parts -contains $d) { " +
		"$n=($parts | Where-Object { $_ -ne $d }) -join ';'; " +
		"[Environment]::SetEnvironmentVariable('Path',$n,'User') }"
	_ = exec.Command("powershell", "-NoProfile", "-Command", ps).Run()
}
