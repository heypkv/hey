package gh

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fixture mirrors a monorepo with cli/-prefixed and foreign tags, a draft,
// and a prerelease — listed in creation order, NOT semver order.
const releasesFixture = `[
  {"tag_name": "cli/v0.2.7", "draft": false, "prerelease": false},
  {"tag_name": "go/v0.9.9", "draft": false, "prerelease": false},
  {"tag_name": "cli/v0.2.10", "draft": false, "prerelease": false},
  {"tag_name": "cli/v0.3.0", "draft": true, "prerelease": false},
  {"tag_name": "cli/v0.4.0", "draft": false, "prerelease": true},
  {"tag_name": "cli/v0.2.9", "draft": false, "prerelease": false},
  {"tag_name": "v9.9.9", "draft": false, "prerelease": false}
]`

func withFixtureServer(t *testing.T, hits *int) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/repos/kitsyai/guten/releases") {
			http.NotFound(w, r)
			return
		}
		if hits != nil {
			*hits++
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(releasesFixture))
	}))
	t.Cleanup(srv.Close)
	old := APIBase
	APIBase = srv.URL
	t.Cleanup(func() { APIBase = old })
}

func TestResolveLatestFiltersPrefixAndPicksSemverMax(t *testing.T) {
	withFixtureServer(t, nil)
	v, err := ResolveLatest("guten", "kitsyai/guten", "cli/", t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	// Not 0.2.7 (creation order trap), not 0.3.0 (draft), not 0.4.0
	// (prerelease), not 9.9.9 (foreign prefix).
	if v != "0.2.10" {
		t.Errorf("resolved %q, want 0.2.10", v)
	}
}

func TestResolveLatestNoPrefix(t *testing.T) {
	withFixtureServer(t, nil)
	v, err := ResolveLatest("x", "kitsyai/guten", "", t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	if v != "9.9.9" {
		t.Errorf("resolved %q, want 9.9.9", v)
	}
}

func TestResolveCacheAvoidsSecondAPICall(t *testing.T) {
	hits := 0
	withFixtureServer(t, &hits)
	stateDir := t.TempDir()
	if _, err := ResolveLatest("guten", "kitsyai/guten", "cli/", stateDir, false); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveLatest("guten", "kitsyai/guten", "cli/", stateDir, false); err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Errorf("API hit %d times, want 1 (second resolve should be cached)", hits)
	}
	// refresh=true must bypass the cache.
	if _, err := ResolveLatest("guten", "kitsyai/guten", "cli/", stateDir, true); err != nil {
		t.Fatal(err)
	}
	if hits != 2 {
		t.Errorf("API hit %d times after refresh, want 2", hits)
	}
}

func TestResolveCacheExpires(t *testing.T) {
	hits := 0
	withFixtureServer(t, &hits)
	stateDir := t.TempDir()
	if _, err := ResolveLatest("guten", "kitsyai/guten", "cli/", stateDir, false); err != nil {
		t.Fatal(err)
	}
	// Age the cache entry beyond the TTL.
	path := filepath.Join(stateDir, resolveFname)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var c resolveCache
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	e := c.Apps["guten"]
	e.Checked = time.Now().Add(-25 * time.Hour)
	c.Apps["guten"] = e
	aged, _ := json.Marshal(c)
	if err := os.WriteFile(path, aged, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveLatest("guten", "kitsyai/guten", "cli/", stateDir, false); err != nil {
		t.Fatal(err)
	}
	if hits != 2 {
		t.Errorf("API hit %d times, want 2 (expired cache should re-resolve)", hits)
	}
}

func TestNoMatchingReleases(t *testing.T) {
	withFixtureServer(t, nil)
	if _, err := ResolveLatest("x", "kitsyai/guten", "nope/", t.TempDir(), false); err == nil {
		t.Error("expected error for prefix with no releases")
	}
}

func TestDownloadURL(t *testing.T) {
	got := DownloadURL("kitsyai/guten", "cli/v0.2.7", "guten_0.2.7_windows_amd64.zip")
	want := "https://github.com/kitsyai/guten/releases/download/cli/v0.2.7/guten_0.2.7_windows_amd64.zip"
	if got != want {
		t.Errorf("DownloadURL = %q, want %q", got, want)
	}
}

func TestSemverParse(t *testing.T) {
	if _, ok := parseSemver("1.2.3"); !ok {
		t.Error("1.2.3 should parse")
	}
	for _, bad := range []string{"1.2", "1.2.3.4", "1.2.x", "", "1.2.-3"} {
		if _, ok := parseSemver(bad); ok {
			t.Errorf("%q should not parse", bad)
		}
	}
	a, _ := parseSemver("0.2.9")
	b, _ := parseSemver("0.2.10")
	if !semverLess(a, b) {
		t.Error("0.2.9 < 0.2.10 (numeric, not lexicographic)")
	}
}
