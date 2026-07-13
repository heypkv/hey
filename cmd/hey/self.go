package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/kitsyai/hey/internal/fetch"
	"github.com/kitsyai/hey/internal/gh"
	"github.com/kitsyai/hey/internal/home"
)

// selfRepo is hey's own release home — this is hey knowing where hey lives, not
// product knowledge.
const selfRepo = "kitsyai/hey"

// cmdSelf handles `hey self update`: fetch the latest hey release, verify its
// checksum, and replace the running binary in place.
func cmdSelf(args []string) error {
	if len(args) != 1 || args[0] != "update" {
		return fmt.Errorf("usage: hey self update")
	}
	stateDir, err := home.StateDir()
	if err != nil {
		return err
	}
	latest, err := gh.ResolveLatest("hey", selfRepo, "", stateDir, true)
	if err != nil {
		return err
	}
	if latest == version {
		fmt.Printf("hey is already the latest (%s)\n", version)
		return nil
	}
	fmt.Printf("updating hey %s -> %s\n", version, latest)

	tag := "v" + latest
	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	asset := fmt.Sprintf("hey_%s_%s_%s.%s", latest, runtime.GOOS, runtime.GOARCH, ext)

	sumsRaw, err := fetch.Checksums(gh.DownloadURL(selfRepo, tag, "checksums.txt"))
	if err != nil {
		return err
	}
	heyHome, err := home.Dir()
	if err != nil {
		return err
	}
	archive, sha, err := fetch.Download(gh.DownloadURL(selfRepo, tag, asset), heyHome)
	if err != nil {
		return err
	}
	defer os.Remove(archive)
	if err := fetch.Verify(asset, sha, fetch.ParseChecksums(sumsRaw), nil); err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp(heyHome, ".selfupdate-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	binName := "hey"
	if runtime.GOOS == "windows" {
		binName = "hey.exe"
	}
	if err := fetch.ExtractBinary(archive, binName, tmpDir); err != nil {
		return err
	}
	newBin := filepath.Join(tmpDir, binName)

	self, err := os.Executable()
	if err != nil {
		return err
	}
	if resolved, e := filepath.EvalSymlinks(self); e == nil {
		self = resolved
	}
	if err := replaceSelf(self, newBin); err != nil {
		return fmt.Errorf("replace hey binary at %s: %w", self, err)
	}
	fmt.Printf("hey updated to %s\n", latest)
	return nil
}

// copyFile copies src to dst with the given mode (used for cross-volume
// replaces where os.Rename can't reach).
func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
