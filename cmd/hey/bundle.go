package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kitsyai/hey/internal/deploy"
	"github.com/kitsyai/hey/internal/home"
)

// bundleMeta records how a deployed bundle was installed so hey can update it,
// re-verify it, and let the user enable/disable it. Stored at
// ~/.hey/apps/<id>/meta.json.
type bundleMeta struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"` // "scope", "url", or "source"
	Scope     string    `json:"scope,omitempty"`
	Channel   string    `json:"channel,omitempty"`
	URL       string    `json:"url,omitempty"`
	Repo      string    `json:"repo,omitempty"` // source installs: owner/repo
	Cred      string    `json:"cred,omitempty"` // source installs: keeper credential name
	Exec      string    `json:"exec,omitempty"` // basename of the installed executable (source installs)
	Current   string    `json:"current"`
	Enabled   bool      `json:"enabled"`
	Untrusted bool      `json:"untrusted,omitempty"` // installed with --allow-untrusted
	Updated   time.Time `json:"updated"`
}

func (m bundleMeta) source() string {
	if m.Kind == "scope" {
		return "@" + m.Scope + "/" + m.ID
	}
	return m.URL
}

// ref reconstructs the install reference so `hey update` can re-resolve it.
func (m bundleMeta) ref() deploy.Ref {
	if m.Kind == "scope" {
		return deploy.Ref{Kind: deploy.RefScoped, Scope: m.Scope, ID: m.ID}
	}
	return deploy.Ref{Kind: deploy.RefManifestURL, ManifestURL: m.URL}
}

func metaPath(id string) (string, error) {
	apps, err := home.AppsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(apps, id, "meta.json"), nil
}

func readMeta(id string) (bundleMeta, bool, error) {
	p, err := metaPath(id)
	if err != nil {
		return bundleMeta{}, false, err
	}
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return bundleMeta{}, false, nil
	}
	if err != nil {
		return bundleMeta{}, false, err
	}
	var m bundleMeta
	if err := json.Unmarshal(data, &m); err != nil {
		return bundleMeta{}, false, fmt.Errorf("parse %s: %w", p, err)
	}
	return m, true, nil
}

func writeMeta(m bundleMeta) error {
	p, err := metaPath(m.ID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, append(data, '\n'), 0o644)
}

// recordBundle writes/updates a bundle's meta after a successful install. It
// preserves an existing enabled flag; new bundles default to enabled.
func recordBundle(m *deploy.Manifest, ref deploy.Ref, o deployOpts) error {
	meta := bundleMeta{ID: m.ID, Current: m.Version, Enabled: true, Untrusted: o.allowUntrusted, Updated: time.Now()}
	if existing, ok, _ := readMeta(m.ID); ok {
		meta.Enabled = existing.Enabled
	}
	switch ref.Kind {
	case deploy.RefScoped:
		meta.Kind = "scope"
		meta.Scope = ref.Scope
		meta.Channel = o.channel
	case deploy.RefManifestURL:
		meta.Kind = "url"
		meta.URL = ref.ManifestURL
	}
	return writeMeta(meta)
}

// bundleDisabled reports whether a bundle id is installed and disabled.
func bundleDisabled(id string) bool {
	m, ok, _ := readMeta(id)
	return ok && !m.Enabled
}

func cmdEnable(args []string) error  { return setEnabled(args, true) }
func cmdDisable(args []string) error { return setEnabled(args, false) }

func setEnabled(args []string, enabled bool) error {
	verb := "enable"
	if !enabled {
		verb = "disable"
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: hey %s <id>", verb)
	}
	id := args[0]
	m, ok, err := readMeta(id)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("no installed bundle %q", id)
	}
	m.Enabled = enabled
	if err := writeMeta(m); err != nil {
		return err
	}
	fmt.Printf("%sd %s\n", verb, id)
	return nil
}

// cmdRemove deletes an installed bundle (or a github-release app) and its cache.
func cmdRemove(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: hey remove <id>")
	}
	name, _ := splitAppRef(args[0])
	appsDir, err := home.AppsDir()
	if err != nil {
		return err
	}
	binDir, err := home.BinDir()
	if err != nil {
		return err
	}
	removed := false
	for _, target := range []string{filepath.Join(appsDir, name), filepath.Join(binDir, name)} {
		if _, err := os.Stat(target); err == nil {
			if err := os.RemoveAll(target); err != nil {
				return err
			}
			removed = true
		}
	}
	if !removed {
		return fmt.Errorf("%q is not installed", name)
	}
	removeShim(name) // best-effort: drop the PATH shim a source install created
	fmt.Printf("removed %s\n", name)
	return nil
}

// updateBundle re-resolves an installed bundle's source and installs the
// channel's current version (a no-op if already current). It carries forward
// the original untrusted consent so a URL bundle doesn't re-prompt.
func updateBundle(id string) error {
	m, ok, err := readMeta(id)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("no installed bundle %q", id)
	}
	prev := m.Current
	o := deployOpts{channel: m.Channel, allowUntrusted: m.Untrusted, timeout: 30 * time.Second}
	if err := installDeployRef(m.ref(), o); err != nil {
		return err
	}
	if now, ok, _ := readMeta(id); ok && now.Current != prev {
		fmt.Printf("%s updated %s -> %s\n", id, prev, now.Current)
	} else {
		fmt.Printf("%s is already up to date (%s)\n", id, prev)
	}
	return nil
}

// isManagedKind reports whether a kind installs a persistent, manageable
// bundle (as opposed to a native installer handed to the OS, or a link).
func isManagedKind(kind string) bool {
	switch kind {
	case deploy.KindArchive, deploy.KindBinary, deploy.KindAppImage:
		return true
	}
	return false
}

// installedBundles lists the ids of deployed bundles (dirs with a meta.json).
func installedBundles() []bundleMeta {
	apps, err := home.AppsDir()
	if err != nil {
		return nil
	}
	entries, err := os.ReadDir(apps)
	if err != nil {
		return nil
	}
	var out []bundleMeta
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if m, ok, _ := readMeta(e.Name()); ok {
			out = append(out, m)
		}
	}
	return out
}
