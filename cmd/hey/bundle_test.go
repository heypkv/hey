package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kitsyai/hey/internal/home"
)

func TestBundleEnableDisableRemove(t *testing.T) {
	t.Setenv("HEY_HOME", t.TempDir())

	// Synthesize an installed bundle: a version dir + meta.
	appsDir, _ := home.AppsDir()
	if err := os.MkdirAll(filepath.Join(appsDir, "demo", "1.0.0"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeMeta(bundleMeta{ID: "demo", Kind: "scope", Scope: "heypkv", Channel: "stable", Current: "1.0.0", Enabled: true}); err != nil {
		t.Fatal(err)
	}

	if err := cmdDisable([]string{"demo"}); err != nil {
		t.Fatal(err)
	}
	if !bundleDisabled("demo") {
		t.Error("demo should be disabled")
	}
	if err := cmdEnable([]string{"demo"}); err != nil {
		t.Fatal(err)
	}
	if bundleDisabled("demo") {
		t.Error("demo should be enabled again")
	}
	if err := cmdDisable([]string{"missing"}); err == nil {
		t.Error("disabling an uninstalled bundle should error")
	}

	if err := cmdRemove([]string{"demo"}); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := readMeta("demo"); ok {
		t.Error("meta should be gone after remove")
	}
	if _, err := os.Stat(filepath.Join(appsDir, "demo")); !os.IsNotExist(err) {
		t.Error("bundle dir should be gone after remove")
	}
	if err := cmdRemove([]string{"demo"}); err == nil {
		t.Error("removing a missing bundle should error")
	}
}

func TestReplaceSelf(t *testing.T) {
	dir := t.TempDir()
	self := filepath.Join(dir, "hey.bin")
	if err := os.WriteFile(self, []byte("OLD"), 0o755); err != nil {
		t.Fatal(err)
	}
	newBin := filepath.Join(dir, "new.bin")
	if err := os.WriteFile(newBin, []byte("NEW-VERSION"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := replaceSelf(self, newBin); err != nil {
		t.Fatalf("replaceSelf: %v", err)
	}
	got, err := os.ReadFile(self)
	if err != nil || string(got) != "NEW-VERSION" {
		t.Fatalf("self not replaced: %q err=%v", got, err)
	}
}
