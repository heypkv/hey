package fetch

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestParseChecksums(t *testing.T) {
	data := []byte(`e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  guten_0.2.7_windows_amd64.zip
aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  guten_0.2.7_linux_amd64.tar.gz
not a checksum line
`)
	sums := ParseChecksums(data)
	if len(sums) != 2 {
		t.Fatalf("parsed %d entries, want 2", len(sums))
	}
	if sums["guten_0.2.7_windows_amd64.zip"] != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Error("wrong sha for zip entry")
	}
}

func TestVerify(t *testing.T) {
	sums := map[string]string{"a.zip": "aa", "b.zip": "bb"}
	if err := Verify("a.zip", "AA", sums, nil); err != nil {
		t.Errorf("case-insensitive match should pass: %v", err)
	}
	if err := Verify("a.zip", "bb", sums, nil); err == nil {
		t.Error("mismatch should fail")
	}
	if err := Verify("missing.zip", "aa", sums, nil); err == nil {
		t.Error("missing checksums entry should fail")
	}
	if err := Verify("a.zip", "aa", sums, &SigSpec{}); err == nil {
		t.Error("signature spec should fail until implemented")
	}
}

func TestDownloadRejectsHTTP(t *testing.T) {
	if _, _, err := Download("http://example.com/x.zip", t.TempDir()); err == nil ||
		!strings.Contains(err.Error(), "https") {
		t.Errorf("plain http must be refused, got %v", err)
	}
}

func TestDownloadComputesSHA(t *testing.T) {
	payload := []byte("hello hey")
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(payload)
	}))
	defer srv.Close()
	oldClient := httpClient
	httpClient = srv.Client() // trusts the test TLS cert; URL is https://
	defer func() { httpClient = oldClient }()

	dir := t.TempDir()
	path, sha, err := Download(srv.URL+"/x.zip", dir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)
	want := sha256.Sum256(payload)
	if sha != hex.EncodeToString(want[:]) {
		t.Errorf("sha = %s, want %s", sha, hex.EncodeToString(want[:]))
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(payload) {
		t.Errorf("downloaded content mismatch: %v", err)
	}
}
