package fetch

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeZip(t *testing.T, entries map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "a.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(content))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return path
}

func makeTarGz(t *testing.T, add func(*tar.Writer)) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "a.tar.gz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	add(tw)
	tw.Close()
	gz.Close()
	f.Close()
	return path
}

func tarFile(t *testing.T, tw *tar.Writer, name, content string) {
	t.Helper()
	if err := tw.WriteHeader(&tar.Header{
		Name: name, Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg,
	}); err != nil {
		t.Fatal(err)
	}
	tw.Write([]byte(content))
}

func TestExtractsOnlyTheBinaryFromZip(t *testing.T) {
	archive := makeZip(t, map[string]string{
		"README.md":  "docs",
		"LICENSE":    "mit",
		"guten.exe":  "binary-bytes",
		"extra/file": "junk",
	})
	dest := t.TempDir()
	if err := ExtractBinary(archive, "guten.exe", dest); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dest)
	if len(entries) != 1 || entries[0].Name() != "guten.exe" {
		t.Errorf("dest should contain exactly guten.exe, got %v", entries)
	}
	data, _ := os.ReadFile(filepath.Join(dest, "guten.exe"))
	if string(data) != "binary-bytes" {
		t.Error("binary content mismatch")
	}
}

func TestExtractsFromTarGz(t *testing.T) {
	archive := makeTarGz(t, func(tw *tar.Writer) {
		tarFile(t, tw, "README.md", "docs")
		tarFile(t, tw, "guten", "elf-bytes")
	})
	dest := t.TempDir()
	// The archive has no .zip suffix and no zip magic → tar.gz path.
	if err := ExtractBinary(archive, "guten", dest); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(dest, "guten"))
	if string(data) != "elf-bytes" {
		t.Error("binary content mismatch")
	}
}

func TestMissingBinaryFails(t *testing.T) {
	archive := makeZip(t, map[string]string{"README.md": "docs"})
	if err := ExtractBinary(archive, "guten.exe", t.TempDir()); err == nil ||
		!strings.Contains(err.Error(), "does not contain") {
		t.Errorf("missing binary should fail, got %v", err)
	}
}

func TestZipSlipRejected(t *testing.T) {
	for _, evil := range []string{"../evil.exe", "..\\evil.exe", "/abs/evil.exe", "sub/../../evil.exe"} {
		archive := makeZip(t, map[string]string{evil: "boom", "guten.exe": "ok"})
		if err := ExtractBinary(archive, "guten.exe", t.TempDir()); err == nil ||
			!strings.Contains(err.Error(), "unsafe path") {
			t.Errorf("entry %q should be rejected, got %v", evil, err)
		}
	}
}

func TestTarSymlinkRejected(t *testing.T) {
	archive := makeTarGz(t, func(tw *tar.Writer) {
		if err := tw.WriteHeader(&tar.Header{
			Name: "guten", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd",
		}); err != nil {
			t.Fatal(err)
		}
	})
	if err := ExtractBinary(archive, "guten", t.TempDir()); err == nil ||
		!strings.Contains(err.Error(), "link") {
		t.Errorf("symlink entry should be rejected, got %v", err)
	}
}

func TestTarDotDotRejected(t *testing.T) {
	archive := makeTarGz(t, func(tw *tar.Writer) {
		tarFile(t, tw, "../guten", "boom")
	})
	if err := ExtractBinary(archive, "guten", t.TempDir()); err == nil ||
		!strings.Contains(err.Error(), "unsafe path") {
		t.Errorf("dot-dot tar entry should be rejected, got %v", err)
	}
}

func TestExtractTreeZipPreservesLayout(t *testing.T) {
	archive := makeZip(t, map[string]string{
		"pgsql/bin/initdb.exe": "init",
		"pgsql/bin/postgres.exe": "server",
		"pgsql/share/postgresql.conf": "config",
		"README":                      "docs",
	})
	dest := t.TempDir()
	if err := ExtractTree(archive, dest); err != nil {
		t.Fatal(err)
	}
	for rel, want := range map[string]string{
		"pgsql/bin/initdb.exe":        "init",
		"pgsql/bin/postgres.exe":      "server",
		"pgsql/share/postgresql.conf": "config",
		"README":                      "docs",
	} {
		got, err := os.ReadFile(filepath.Join(dest, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		if string(got) != want {
			t.Errorf("%s = %q, want %q", rel, got, want)
		}
	}
}

func TestExtractTreeTarGzPreservesLayout(t *testing.T) {
	archive := makeTarGz(t, func(tw *tar.Writer) {
		tarFile(t, tw, "pgsql/bin/initdb", "init")
		tarFile(t, tw, "pgsql/share/x.conf", "config")
	})
	dest := t.TempDir()
	if err := ExtractTree(archive, dest); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, filepath.FromSlash("pgsql/bin/initdb")))
	if err != nil || string(got) != "init" {
		t.Errorf("initdb = %q err=%v", got, err)
	}
}

func TestExtractTreeZipSlipRejected(t *testing.T) {
	for _, evil := range []string{"../evil", "..\\evil", "/abs/evil", "sub/../../evil"} {
		archive := makeZip(t, map[string]string{evil: "boom", "ok": "fine"})
		if err := ExtractTree(archive, t.TempDir()); err == nil {
			t.Errorf("entry %q should be rejected", evil)
		}
	}
}

func TestExtractTreeTarSymlinkRejected(t *testing.T) {
	archive := makeTarGz(t, func(tw *tar.Writer) {
		if err := tw.WriteHeader(&tar.Header{
			Name: "link", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd",
		}); err != nil {
			t.Fatal(err)
		}
	})
	if err := ExtractTree(archive, t.TempDir()); err == nil ||
		!strings.Contains(err.Error(), "link") {
		t.Errorf("symlink entry should be rejected, got %v", err)
	}
}
