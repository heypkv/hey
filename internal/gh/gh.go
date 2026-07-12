// Package gh resolves app versions against GitHub Releases and constructs
// download URLs. It never uses /releases/latest (blind to monorepo tag
// prefixes); it lists releases, filters by prefix, and picks the semver max.
// Successful resolutions are cached on disk (24h TTL) because unauthenticated
// GitHub API calls are limited to 60/hour/IP.
package gh

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// APIBase is overridable for tests.
var APIBase = "https://api.github.com"

// DownloadBase is overridable for tests.
var DownloadBase = "https://github.com"

const (
	cacheTTL     = 24 * time.Hour
	maxAPIBytes  = 4 << 20 // 4 MB release listing cap
	resolveFname = "resolve.json"
)

type release struct {
	TagName    string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
}

// ResolveLatest returns the newest released version (without the leading "v")
// for repo whose tags match tagPrefix+"v". Results are cached under stateDir
// keyed by app; pass refresh=true to bypass the cache (hey update).
func ResolveLatest(app, repo, tagPrefix, stateDir string, refresh bool) (string, error) {
	if !refresh {
		if v, ok := cachedVersion(stateDir, app); ok {
			return v, nil
		}
	}
	v, err := resolveFromAPI(repo, tagPrefix)
	if err != nil {
		return "", err
	}
	saveCachedVersion(stateDir, app, v)
	return v, nil
}

func resolveFromAPI(repo, tagPrefix string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/releases?per_page=100", APIBase, repo)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if tok := os.Getenv("GH_TOKEN"); tok == "" {
		if tok = os.Getenv("GITHUB_TOKEN"); tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
	} else {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("list releases for %s: %w", repo, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden && resp.Header.Get("X-RateLimit-Remaining") == "0" {
		return "", fmt.Errorf("GitHub API rate limit exhausted; set GH_TOKEN or retry later")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("list releases for %s: HTTP %d", repo, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxAPIBytes))
	if err != nil {
		return "", err
	}
	var releases []release
	if err := json.Unmarshal(body, &releases); err != nil {
		return "", fmt.Errorf("parse releases for %s: %w", repo, err)
	}

	prefix := tagPrefix + "v"
	best := ""
	var bestV [3]int
	for _, rel := range releases {
		if rel.Draft || rel.Prerelease {
			continue
		}
		if !strings.HasPrefix(rel.TagName, prefix) {
			continue
		}
		ver := strings.TrimPrefix(rel.TagName, prefix)
		v, ok := parseSemver(ver)
		if !ok {
			continue
		}
		// API order is creation date, not semver — always max-compare.
		if best == "" || semverLess(bestV, v) {
			best, bestV = ver, v
		}
	}
	if best == "" {
		return "", fmt.Errorf("no releases matching tag prefix %q in %s", prefix, repo)
	}
	return best, nil
}

func parseSemver(s string) ([3]int, bool) {
	var v [3]int
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return v, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return v, false
		}
		v[i] = n
	}
	return v, true
}

func semverLess(a, b [3]int) bool {
	for i := 0; i < 3; i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

// DownloadURL builds the (API-rate-limit-free) release asset URL.
func DownloadURL(repo, tag, asset string) string {
	return fmt.Sprintf("%s/%s/releases/download/%s/%s", DownloadBase, repo, tag, asset)
}

// --- resolve cache ---

type resolveCache struct {
	Apps map[string]resolveEntry `json:"apps"`
}

type resolveEntry struct {
	Version string    `json:"version"`
	Checked time.Time `json:"checked"`
}

func cachedVersion(stateDir, app string) (string, bool) {
	data, err := os.ReadFile(filepath.Join(stateDir, resolveFname))
	if err != nil {
		return "", false
	}
	var c resolveCache
	if err := json.Unmarshal(data, &c); err != nil {
		return "", false
	}
	e, ok := c.Apps[app]
	if !ok || time.Since(e.Checked) > cacheTTL {
		return "", false
	}
	return e.Version, true
}

func saveCachedVersion(stateDir, app, version string) {
	path := filepath.Join(stateDir, resolveFname)
	c := resolveCache{Apps: map[string]resolveEntry{}}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &c)
		if c.Apps == nil {
			c.Apps = map[string]resolveEntry{}
		}
	}
	c.Apps[app] = resolveEntry{Version: version, Checked: time.Now()}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, path) // best-effort cache; failure just means a re-resolve
}
