package main

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kitsyai/hey/internal/gh"
	"github.com/kitsyai/hey/internal/home"
	"github.com/kitsyai/hey/internal/source"
)

// TestBuddySourceInstall exercises the full source-install path against a
// synthetic GitHub contents API: fetch hey.json, fetch the platform prebuilt,
// verify its sha256, install it, record the bundle, and confirm the recorded
// bundle runs its checked-in executable directly (the `boss …` path).
func TestBuddySourceInstall(t *testing.T) {
	payload := []byte("#!fake-boss-binary\n")
	sum := fmt.Sprintf("%x", sha256.Sum256(payload))
	manifest := fmt.Sprintf(`{
	  "hey_manifest": "hey.source.v1",
	  "id": "boss",
	  "version": "9.9.9",
	  "prebuilt": { %q: {"path": "bin/boss-native", "sha256": %q} }
	}`, source.Platform(), sum)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/contents/hey.json"):
			fmt.Fprint(w, manifest)
		case strings.HasSuffix(r.URL.Path, "/contents/bin/boss-native"):
			w.Write(payload)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	oldBase := gh.APIBase
	gh.APIBase = srv.URL
	defer func() { gh.APIBase = oldBase }()
	t.Setenv("HEY_HOME", t.TempDir())

	if err := buddySourceInstall("kyive/boss", ""); err != nil {
		t.Fatalf("buddySourceInstall: %v", err)
	}

	// meta recorded as a source bundle with an exec name.
	m, ok, err := readMeta("boss")
	if err != nil || !ok {
		t.Fatalf("readMeta: ok=%v err=%v", ok, err)
	}
	if m.Kind != "source" || m.Current != "9.9.9" || m.Exec == "" {
		t.Fatalf("bad meta: %+v", m)
	}

	// the installed executable holds the fetched bytes.
	dir, err := home.DeployAppDir("boss", "9.9.9")
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, m.Exec))
	if err != nil {
		t.Fatalf("read installed exec: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("installed exec content mismatch")
	}
}

// TestBuddySourceUpdate proves update semantics: re-running with the same
// manifest is a no-op; a new binary (new sha) at the same version reinstalls;
// and updateBundle routes source bundles through the same path.
func TestBuddySourceUpdate(t *testing.T) {
	payload := []byte("boss-v1\n")
	var manifest func() string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/contents/hey.json") {
			fmt.Fprint(w, manifest())
			return
		}
		w.Write(payload)
	}))
	defer srv.Close()
	setManifest := func() string {
		return fmt.Sprintf(`{"hey_manifest":"hey.source.v1","id":"boss","version":"1.0.0",
		  "prebuilt":{%q:{"path":"bin/boss","sha256":%q}}}`,
			source.Platform(), fmt.Sprintf("%x", sha256.Sum256(payload)))
	}
	manifest = setManifest

	oldBase := gh.APIBase
	gh.APIBase = srv.URL
	defer func() { gh.APIBase = oldBase }()
	t.Setenv("HEY_HOME", t.TempDir())

	if err := buddySourceInstall("kyive/boss", ""); err != nil {
		t.Fatalf("install: %v", err)
	}

	// Same manifest → both update spellings are a no-op (version + sha unchanged).
	if err := updateBundle("boss"); err != nil {
		t.Fatalf("update (no change): %v", err)
	}
	if err := buddyUpdate([]string{"boss"}); err != nil {
		t.Fatalf("buddy update (no change): %v", err)
	}

	// New binary at the same version → reinstall picks it up.
	payload = []byte("boss-v2-fixed\n")
	manifest = setManifest // recomputes the sha for the new payload
	if err := updateBundle("boss"); err != nil {
		t.Fatalf("update (changed): %v", err)
	}
	dir, _ := home.DeployAppDir("boss", "1.0.0")
	m, _, _ := readMeta("boss")
	got, _ := os.ReadFile(filepath.Join(dir, m.Exec))
	if string(got) != string(payload) {
		t.Fatalf("update did not replace binary: got %q", got)
	}
}

// TestBuddySourceInstallChecksumMismatch rejects a tampered binary.
func TestBuddySourceInstallChecksumMismatch(t *testing.T) {
	manifest := fmt.Sprintf(`{"hey_manifest":"hey.source.v1","id":"boss","version":"1.0.0",
	  "prebuilt":{%q:{"path":"bin/boss-native","sha256":"deadbeef"}}}`, source.Platform())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/hey.json") {
			fmt.Fprint(w, manifest)
			return
		}
		w.Write([]byte("actual-bytes"))
	}))
	defer srv.Close()

	oldBase := gh.APIBase
	gh.APIBase = srv.URL
	defer func() { gh.APIBase = oldBase }()
	t.Setenv("HEY_HOME", t.TempDir())

	err := buddySourceInstall("kyive/boss", "")
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch, got %v", err)
	}
}
