// Package svc provisions and manages local services (postgres, caches, object
// storage) from data-only *packs* run by drivers that ship in the hey binary.
// See docs/pack-manifest-v0.md. v0 implements one driver, archive-exec, which
// contains zero service-specific logic — everything about postgres lives in
// its pack, never here.
package svc

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

//go:embed packs.json
var defaultPacksJSON []byte

// PacksVersion is the pack-manifest format version this build understands.
const PacksVersion = 0

// DriverArchiveExec is the only driver v0 ships.
const DriverArchiveExec = "archive-exec"

const maxPacksBytes = 1 << 20 // 1 MB

// PackSet is a collection of packs, embedded or overridden (same format).
type PackSet struct {
	HeyPacks int             `json:"hey_packs"`
	Packs    map[string]Pack `json:"packs"`
}

// Pack describes one service as data. The driver interprets it.
type Pack struct {
	Pack     string             `json:"pack"`
	Driver   string             `json:"driver"`
	Kind     string             `json:"kind,omitempty"`
	Versions map[string]Version `json:"versions"`
	Init     []string           `json:"init,omitempty"`
	Start    string             `json:"start"`
	Ready    Ready              `json:"ready"`
	Conn     string             `json:"conn,omitempty"`
	Stop     StopSpec           `json:"stop"`
}

// Version holds the per-platform artifacts for one pack version.
type Version struct {
	Artifacts map[string]Artifact `json:"artifacts"`
}

// Artifact is one platform's downloadable archive.
type Artifact struct {
	URL       string `json:"url"`
	SHA256    string `json:"sha256"`
	BinSubdir string `json:"bin_subdir,omitempty"`
}

// Ready is a health check: exactly one of TCP or Command.
type Ready struct {
	TCP            string `json:"tcp,omitempty"`
	Command        string `json:"command,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
}

// StopSpec describes graceful shutdown; the driver force-kills as a last resort.
type StopSpec struct {
	Command      string `json:"command,omitempty"`
	Signal       string `json:"signal,omitempty"`
	GraceSeconds int    `json:"grace_seconds,omitempty"`
}

// LoadPacks resolves the effective pack set. Precedence: HEY_PACKS override
// (path or https URL), then <heyHome>/packs.json, then the embedded default.
func LoadPacks(heyHome string) (*PackSet, error) {
	if override := os.Getenv("HEY_PACKS"); override != "" {
		data, err := readPacksOverride(override)
		if err != nil {
			return nil, err
		}
		return parsePacks(data, override)
	}
	userFile := filepath.Join(heyHome, "packs.json")
	if data, err := os.ReadFile(userFile); err == nil {
		return parsePacks(data, userFile)
	}
	return parsePacks(defaultPacksJSON, "embedded default")
}

func readPacksOverride(src string) ([]byte, error) {
	if strings.HasPrefix(src, "http://") {
		return nil, fmt.Errorf("packs URL must be https: %s", src)
	}
	if strings.HasPrefix(src, "https://") {
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Get(src)
		if err != nil {
			return nil, fmt.Errorf("fetch packs %s: %w", src, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("fetch packs %s: HTTP %d", src, resp.StatusCode)
		}
		return io.ReadAll(io.LimitReader(resp.Body, maxPacksBytes))
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return nil, fmt.Errorf("read packs: %w", err)
	}
	return data, nil
}

func parsePacks(data []byte, from string) (*PackSet, error) {
	var ps PackSet
	if err := json.Unmarshal(data, &ps); err != nil {
		return nil, fmt.Errorf("parse packs (%s): %w", from, err)
	}
	if ps.HeyPacks > PacksVersion {
		return nil, fmt.Errorf("packs (%s) use format v%d; this hey only understands v%d — update hey", from, ps.HeyPacks, PacksVersion)
	}
	for name, p := range ps.Packs {
		if err := validatePack(name, p); err != nil {
			return nil, fmt.Errorf("packs (%s): %w", from, err)
		}
	}
	return &ps, nil
}

func validatePack(name string, p Pack) error {
	if strings.ContainsAny(name, `/\@ `) || name == "" {
		return fmt.Errorf("invalid pack name %q", name)
	}
	if p.Pack != name {
		return fmt.Errorf("pack %q: field \"pack\" is %q, must match its key", name, p.Pack)
	}
	if p.Driver != DriverArchiveExec {
		return fmt.Errorf("pack %q: unsupported driver %q (this hey only supports %q)", name, p.Driver, DriverArchiveExec)
	}
	if len(p.Versions) == 0 {
		return fmt.Errorf("pack %q: no versions", name)
	}
	if p.Start == "" {
		return fmt.Errorf("pack %q: missing start command", name)
	}
	if p.Ready.TCP == "" && p.Ready.Command == "" {
		return fmt.Errorf("pack %q: ready needs a tcp or command check", name)
	}
	for ver, v := range p.Versions {
		if len(v.Artifacts) == 0 {
			return fmt.Errorf("pack %q version %q: no artifacts", name, ver)
		}
		for plat, a := range v.Artifacts {
			if !strings.HasPrefix(a.URL, "https://") {
				return fmt.Errorf("pack %q %s/%s: artifact url must be https", name, ver, plat)
			}
			if len(a.SHA256) != 64 {
				return fmt.Errorf("pack %q %s/%s: sha256 must be 64 hex chars", name, ver, plat)
			}
		}
	}
	return nil
}

// Get returns the named pack.
func (ps *PackSet) Get(name string) (Pack, error) {
	p, ok := ps.Packs[name]
	if !ok {
		return Pack{}, fmt.Errorf("unknown pack %q — packs available: %s", name, strings.Join(ps.Names(), ", "))
	}
	return p, nil
}

// Names returns the pack names, sorted.
func (ps *PackSet) Names() []string {
	names := make([]string, 0, len(ps.Packs))
	for n := range ps.Packs {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// LatestVersion returns the highest version string of the pack (semver-ish,
// falling back to lexical). Explicit pins skip this.
func (p Pack) LatestVersion() string {
	vers := make([]string, 0, len(p.Versions))
	for v := range p.Versions {
		vers = append(vers, v)
	}
	sort.Sort(byVersion(vers))
	if len(vers) == 0 {
		return ""
	}
	return vers[len(vers)-1]
}

// Artifact resolves the artifact for version on the current platform.
func (p Pack) Artifact(version string) (Artifact, string, error) {
	v, ok := p.Versions[version]
	if !ok {
		return Artifact{}, "", fmt.Errorf("pack %q has no version %q", p.Pack, version)
	}
	key := runtime.GOOS + "_" + runtime.GOARCH
	a, ok := v.Artifacts[key]
	if !ok {
		return Artifact{}, "", fmt.Errorf("pack %q %s has no artifact for %s", p.Pack, version, key)
	}
	return a, key, nil
}
